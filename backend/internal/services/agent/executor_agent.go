package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"backend/internal/config"
)

// ExecutorAgent runs commands and tests
type ExecutorAgent struct {
	baseAgent
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Command  string        `json:"command"`
	Output   string        `json:"output"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// NewExecutorAgent creates a new executor agent
func NewExecutorAgent() *ExecutorAgent {
	agent := &ExecutorAgent{}

	// Try to load configuration from agent_configs.json
	configPath := "plans/agent_configs.json"
	configData, err := os.ReadFile(configPath)
	if err == nil {
		// Parse JSON data
		var configs []AgentConfig
		if err := json.Unmarshal(configData, &configs); err == nil {
			// Find the executor agent configuration
			for _, config := range configs {
				if config.Name == "executor" {
					agent.name = config.Name
					agent.role = config.Role
					agent.description = config.Description
					agent.prompt = config.Prompt
					return agent
				}
			}
		}
	}

	// Fallback to default values if configuration loading fails
	agent.name = "executor"
	agent.role = "executor"
	agent.description = "Executes commands and runs tests"
	agent.prompt = `You are a DevOps engineer. Your job is to execute commands safely and report results.

When executing commands:
1. Verify the command is safe to run
2. Check for required dependencies
3. Handle errors gracefully
4. Report clear output

You can execute:
- Package installation (npm install, pip install, etc.)
- Build commands (npm run build, make, etc.)
- Test commands (npm test, pytest, go test, etc.)
- File operations (ls, cat, grep, etc.)
- Git commands
- Web requests (curl, wget)

MCP TOOL CALLING:
To use MCP tools, format your command as: mcp://server-name/tool-name
Example: mcp://search/web-search

Always check the exit code and report any errors.`

	return agent
}

// Execute runs the executor agent
func (a *ExecutorAgent) Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error) {
	// Parse command from input
	command := a.parseCommand(input)

	fmt.Printf("[Executor] Parsed command: %s\n", command)

	// Check if this is an MCP tool call request
	if strings.Contains(command, "mcp://") || strings.HasPrefix(command, "{") {
		return a.executeMCPTool(command, mcpManager)
	}

	// Validate command safety
	if !a.isCommandSafe(command) {
		return "", fmt.Errorf("command '%s' is not allowed for security reasons", command)
	}

	// Execute command
	result, err := a.executeCommand(ctx, command)

	// Always log the output for debugging
	fmt.Printf("[Executor] Command output: %s\n", result.Output)

	if err != nil {
		fmt.Printf("[Executor] Command failed: %v, Output: %s\n", err, result.Output)
		return "", fmt.Errorf("command execution failed: %v\nOutput: %s", err, result.Output)
	}

	// Format result
	fmt.Printf("[Executor] Command succeeded, output length: %d\n", len(result.Output))
	return a.formatResult(result), nil
}

// executeMCPTool executes an MCP tool call
func (a *ExecutorAgent) executeMCPTool(command string, mcpManager *MCPManager) (string, error) {
	// Parse MCP tool call format: mcp://server/tool or JSON
	if strings.HasPrefix(command, "mcp://") {
		parts := strings.Split(strings.TrimPrefix(command, "mcp://"), "/")
		if len(parts) >= 2 {
			serverName := parts[0]
			toolName := parts[1]

			// Execute the tool using CallTool
			result, err := mcpManager.CallTool(serverName, toolName, map[string]interface{}{})
			if err != nil {
				return "", fmt.Errorf("MCP tool execution failed: %v", err)
			}
			return result, nil
		}
	}

	return "", fmt.Errorf("invalid MCP tool call format: %s", command)
}

// parseCommand extracts the command from input
func (a *ExecutorAgent) parseCommand(input string) string {
	// Remove common prefixes
	input = strings.TrimSpace(input)
	prefixes := []string{
		"run ", "execute ", "command: ", "cmd: ",
		"Run ", "Execute ", "Command: ", "Cmd: ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(input, prefix) {
			input = strings.TrimPrefix(input, prefix)
			break
		}
	}

	// Sanitize command - replace smart quotes with regular quotes
	replacements := map[string]string{
		"\u2018": "'",  // left single quote
		"\u2019": "'",  // right single quote
		"\u201C": "\"", // left double quote
		"\u201D": "\"", // right double quote
		"\u2013": "-",  // en dash
		"\u2014": "-",  // em dash
	}
	for old, new := range replacements {
		input = strings.ReplaceAll(input, old, new)
	}

	// Take only the first line (in case there's description after the command)
	if idx := strings.IndexAny(input, "\n\r"); idx != -1 {
		input = input[:idx]
	}

	// Ensure the command has balanced quotes
	// Count single quotes
	singleQuotes := strings.Count(input, "'")
	if singleQuotes%2 != 0 {
		// If odd number of single quotes, escape the last one
		lastQuoteIdx := strings.LastIndex(input, "'")
		if lastQuoteIdx != -1 {
			input = input[:lastQuoteIdx] + "\\'" + input[lastQuoteIdx+1:]
		}
	}

	// Count double quotes
	doubleQuotes := strings.Count(input, "\"")
	if doubleQuotes%2 != 0 {
		// If odd number of double quotes, escape the last one
		lastQuoteIdx := strings.LastIndex(input, "\"")
		if lastQuoteIdx != -1 {
			input = input[:lastQuoteIdx] + "\\\"" + input[lastQuoteIdx+1:]
		}
	}

	return strings.TrimSpace(input)
}

// isCommandSafe checks if a command is safe to execute
func (a *ExecutorAgent) isCommandSafe(command string) bool {
	// Block dangerous commands
	dangerousPatterns := []string{
		"rm -rf /", "rm -rf /*", "rm -rf ~", "rm -rf $HOME",
		"dd if=/dev/zero", ":(){ :|:& };:", "> /dev/sda",
		"mkfs", "fdisk", "format",
		"curl.*|.*sh", "wget.*|.*sh",
		"sudo rm", "sudo dd",
	}

	lowerCmd := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCmd, pattern) {
			return false
		}
	}

	// Handle compound commands separated by && or ||
	cmds := strings.FieldsFunc(lowerCmd, func(r rune) bool {
		return r == '&' || r == '|'
	})
	if len(cmds) > 1 {
		// Check each part of the compound command
		for _, cmd := range cmds {
			cmd = strings.TrimSpace(cmd)
			if cmd == "" {
				continue
			}
			// Recursively check each part
			if !a.isCommandSafe(cmd) {
				return false
			}
		}
		return true
	}

	// Allow common safe commands
	safeCommands := []string{
		"npm", "yarn", "pnpm", "pip", "pip3", "python", "python3",
		"go", "cargo", "gradle", "maven", "mvn",
		"docker", "docker-compose", "kubectl",
		"git", "ls", "cat", "grep", "find", "head", "tail",
		"mkdir", "touch", "cp", "mv", "rm", "chmod",
		"echo", "printenv", "which", "pwd",
		"make", "cmake", "gcc", "g++", "clang",
		"pytest", "jest", "mocha", "cargo test", "go test",
		"node", "deno", "bun",
		"curl", "wget", "jq",
		"virtualenv", "python3 -m venv",
		"pip3 install --break-system-packages",
		// Allow apk commands for package management in Docker containers
		"apk update", "apk add",
	}

	for _, safe := range safeCommands {
		if strings.HasPrefix(lowerCmd, safe+" ") || lowerCmd == safe {
			return true
		}
	}

	// Allow file operations in workspace
	if strings.HasPrefix(lowerCmd, "cd ") ||
		strings.HasPrefix(lowerCmd, "ls ") ||
		strings.HasPrefix(lowerCmd, "cat ") ||
		strings.HasPrefix(lowerCmd, "grep ") {
		return true
	}

	// Default deny
	return false
}

// executeCommand runs a command and returns the result
func (a *ExecutorAgent) executeCommand(ctx *TaskContext, command string) (*CommandResult, error) {
	result := &CommandResult{
		Command: command,
	}

	// Determine shell based on OS
	var cmd *exec.Cmd
	if config.GlobalConfig.OS == "windows" {
		cmd = exec.CommandContext(ctx.Context, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx.Context, "sh", "-c", command)
	}

	// Set working directory
	if ctx.WorkingDir != "" {
		cmd.Dir = ctx.WorkingDir
	}

	// Set environment
	cmd.Env = a.buildEnvironment(ctx)

	// Execute with timeout
	start := time.Now()
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)
	result.Output = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Error = err.Error()
		return result, err
	}

	result.ExitCode = 0
	return result, nil
}

// buildEnvironment sets up the command environment
func (a *ExecutorAgent) buildEnvironment(ctx *TaskContext) []string {
	env := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=" + ctx.WorkingDir,
		"PWD=" + ctx.WorkingDir,
	}

	// Add language-specific environment
	env = append(env, "NODE_ENV=development")
	env = append(env, "PYTHONUNBUFFERED=1")
	env = append(env, "GOPROXY=https://proxy.golang.org,direct")
	env = append(env, "CARGO_NET_OFFLINE=false")

	return env
}

// formatResult formats the command result for output
func (a *ExecutorAgent) formatResult(result *CommandResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Command: %s\n", result.Command))
	sb.WriteString(fmt.Sprintf("Exit Code: %d\n", result.ExitCode))
	sb.WriteString(fmt.Sprintf("Duration: %v\n", result.Duration))

	if result.Output != "" {
		sb.WriteString("\nOutput:\n")
		// Limit output length
		output := result.Output
		if len(output) > 5000 {
			output = output[:2500] + "\n... (truncated) ...\n" + output[len(output)-2500:]
		}
		sb.WriteString(output)
	}

	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("\nError: %s\n", result.Error))
	}

	return sb.String()
}

// DetectLanguage detects the programming language from files
func (a *ExecutorAgent) DetectLanguage(files map[string]string) string {
	extensions := make(map[string]int)

	for path := range files {
		ext := getFileExtension(path)
		extensions[ext]++
	}

	// Find most common extension
	var maxExt string
	var maxCount int
	for ext, count := range extensions {
		if count > maxCount {
			maxCount = count
			maxExt = ext
		}
	}

	// Map to language
	langMap := map[string]string{
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".go":    "go",
		".java":  "java",
		".rs":    "rust",
		".cpp":   "cpp",
		".c":     "c",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
	}

	if lang, ok := langMap[maxExt]; ok {
		return lang
	}
	return "unknown"
}

// getFileExtension extracts file extension
func getFileExtension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}

// GetInstallCommand returns the install command for a language
func (a *ExecutorAgent) GetInstallCommand(language string) string {
	commands := map[string]string{
		"python":     "pip install -r requirements.txt",
		"javascript": "npm install",
		"typescript": "npm install",
		"go":         "go mod download",
		"rust":       "cargo build",
		"java":       "mvn install",
		"ruby":       "bundle install",
		"php":        "composer install",
	}

	if cmd, ok := commands[language]; ok {
		return cmd
	}
	return ""
}

// GetTestCommand returns the test command for a language
func (a *ExecutorAgent) GetTestCommand(language string) string {
	commands := map[string]string{
		"python":     "pytest",
		"javascript": "npm test",
		"typescript": "npm test",
		"go":         "go test ./...",
		"rust":       "cargo test",
		"java":       "mvn test",
		"ruby":       "bundle exec rspec",
		"php":        "vendor/bin/phpunit",
	}

	if cmd, ok := commands[language]; ok {
		return cmd
	}
	return ""
}

// GetBuildCommand returns the build command for a language
func (a *ExecutorAgent) GetBuildCommand(language string) string {
	commands := map[string]string{
		"python":     "python setup.py build",
		"javascript": "npm run build",
		"typescript": "npm run build",
		"go":         "go build",
		"rust":       "cargo build --release",
		"java":       "mvn package",
		"c":          "make",
		"cpp":        "make",
	}

	if cmd, ok := commands[language]; ok {
		return cmd
	}
	return ""
}

// ParseTestResults parses test output to extract results
func (a *ExecutorAgent) ParseTestResults(output string, language string) TestSummary {
	summary := TestSummary{}

	switch language {
	case "python":
		// Parse pytest output
		if strings.Contains(output, "passed") {
			fmt.Sscanf(output, "%d passed", &summary.Passed)
		}
		if strings.Contains(output, "failed") {
			fmt.Sscanf(output, "%d failed", &summary.Failed)
		}
		if strings.Contains(output, "error") {
			fmt.Sscanf(output, "%d error", &summary.Errors)
		}

	case "go":
		// Parse go test output
		if strings.Contains(output, "PASS") {
			summary.Passed++
		}
		if strings.Contains(output, "FAIL") {
			summary.Failed++
		}

	case "javascript", "typescript":
		// Parse Jest output
		if strings.Contains(output, "Tests:") {
			// Extract numbers from "Tests: 5 passed, 1 failed"
			parts := strings.Split(output, "Tests:")
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%d passed", &summary.Passed)
				fmt.Sscanf(parts[1], "%d failed", &summary.Failed)
			}
		}
	}

	summary.Total = summary.Passed + summary.Failed + summary.Errors + summary.Skipped
	return summary
}

// TestSummary represents test execution summary
type TestSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Errors  int `json:"errors"`
	Skipped int `json:"skipped"`
}

// String returns a string representation of test summary
func (s TestSummary) String() string {
	return fmt.Sprintf("Tests: %d total, %d passed, %d failed, %d errors, %d skipped",
		s.Total, s.Passed, s.Failed, s.Errors, s.Skipped)
}
