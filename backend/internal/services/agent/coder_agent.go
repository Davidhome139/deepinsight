package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CoderAgent generates and modifies code
type CoderAgent struct {
	baseAgent
}

// CodeBlock represents a generated code file
type CodeBlock struct {
	FilePath string `json:"file_path"`
	Language string `json:"language"`
	Code     string `json:"code"`
}

// NewCoderAgent creates a new coder agent
func NewCoderAgent() *CoderAgent {
	agent := &CoderAgent{}

	// Try to load configuration from agent_configs.json
	configPath := "plans/agent_configs.json"
	configData, err := os.ReadFile(configPath)
	if err == nil {
		// Parse JSON data
		var configs []AgentConfig
		if err := json.Unmarshal(configData, &configs); err == nil {
			// Find the coder agent configuration
			for _, config := range configs {
				if config.Name == "coder" {
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
	agent.name = "coder"
	agent.role = "coder"
	agent.description = "Generates and modifies code files"
	agent.prompt = `You are an expert software developer. Your job is to write clean, production-ready code.

When writing code:
1. Follow best practices for the target language
2. Include proper error handling
3. Add type hints/annotations where appropriate
4. Write self-documenting code with meaningful comments
5. Ensure security best practices
6. Consider edge cases

CRITICAL: You MUST output code in the exact format below, or no files will be created:

### File: FILENAME.py
[CODE START]python
# Your actual Python code here
def main():
    print("Hello")
[CODE END]

### File: FILENAME.js
[CODE START]javascript
// Your actual JavaScript code here
function main() {
    console.log("Hello");
}
[CODE END]

IMPORTANT: Replace [CODE START] with three backticks and [CODE END] with three backticks.

FILENAME CONSISTENCY RULES:
- If the task description or plan specifies a filename, you MUST use that exact filename
- If you need to generate a new filename, ensure it matches any corresponding execute commands in the plan
- Never change filenames from what is specified in the task or plan - this will cause execution failures`

	return agent
}

// Execute runs the coder agent
func (a *CoderAgent) Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error) {
	// Search for code examples via MCP
	examples, err := a.searchCodeExamples(mcpManager, input)
	if err == nil && len(examples) > 0 {
		// Add examples to context
		input = a.addExamplesToInput(input, examples)
	}

	// Enhance input with existing files
	enhancedInput := a.enhanceInput(ctx, input)

	// Call LLM to generate code
	response, err := a.callLLM(enhancedInput)
	if err != nil {
		return "", err
	}

	// Extract code blocks
	codeBlocks := a.extractCodeBlocks(response)
	if len(codeBlocks) == 0 {
		return "", fmt.Errorf("no code blocks found in response")
	}

	// Write files
	for _, block := range codeBlocks {
		if err := a.writeFile(ctx, block); err != nil {
			return "", fmt.Errorf("failed to write file %s: %v", block.FilePath, err)
		}
	}

	return fmt.Sprintf("Generated %d files", len(codeBlocks)), nil
}

// searchCodeExamples searches for relevant code examples
func (a *CoderAgent) searchCodeExamples(mcpManager *MCPManager, query string) ([]CodeExample, error) {
	if mcpManager == nil {
		return nil, fmt.Errorf("MCP manager not available")
	}

	// Use web search MCP if available
	return mcpManager.SearchCodeExamples(query)
}

// addExamplesToInput adds code examples to the input
func (a *CoderAgent) addExamplesToInput(input string, examples []CodeExample) string {
	var sb strings.Builder
	sb.WriteString(input)
	sb.WriteString("\n\nRelevant code examples:\n")

	for i, ex := range examples {
		sb.WriteString(fmt.Sprintf("\n// Example %d from %s\n", i+1, ex.Source))
		sb.WriteString(fmt.Sprintf("// Language: %s\n", ex.Language))
		sb.WriteString(ex.Code)
		sb.WriteString("\n")
	}

	return sb.String()
}

// enhanceInput adds context to the input
func (a *CoderAgent) enhanceInput(ctx *TaskContext, input string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Task: %s\n\n", input))

	if len(ctx.Files) > 0 {
		sb.WriteString("Existing files:\n")
		for path, content := range ctx.Files {
			sb.WriteString(fmt.Sprintf("\n### %s\n", path))
			// Truncate long files
			if len(content) > 1000 {
				sb.WriteString(content[:1000])
				sb.WriteString("\n... (truncated)\n")
			} else {
				sb.WriteString(content)
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(a.prompt)

	return sb.String()
}

// callLLM calls the language model
func (a *CoderAgent) callLLM(input string) (string, error) {
	// Use the base agent's CallLLM method
	return a.CallLLM(input)
}

// OLD callLLM - replaced with real LLM integration
func (a *CoderAgent) oldCallLLM(input string) (string, error) {
	// This would integrate with your LLM client
	// For now, return a mock response
	return fmt.Sprintf(`### File: main.py
���python
# Generated code for: %s

def main():
    print("Hello, World!")

if __name__ == "__main__":
    main()
���`, input[:50]), nil
}

// extractCodeBlocks extracts code blocks from the response
func (a *CoderAgent) extractCodeBlocks(response string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(response, "\n")

	var currentBlock *CodeBlock
	var inCodeBlock bool
	var codeLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for file header
		if strings.HasPrefix(trimmed, "### File:") {
			// Save previous block if exists
			if currentBlock != nil && len(codeLines) > 0 {
				currentBlock.Code = strings.Join(codeLines, "\n")
				blocks = append(blocks, *currentBlock)
			}

			// Start new block
			filePath := strings.TrimSpace(strings.TrimPrefix(trimmed, "### File:"))
			currentBlock = &CodeBlock{
				FilePath: filePath,
				Language: "plaintext",
			}
			codeLines = []string{}
			inCodeBlock = false
			continue
		}

		// Check for code block start (three backticks)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "[CODE START]") {
			if !inCodeBlock {
				// Opening - extract language
				var lang string
				if strings.HasPrefix(trimmed, "```") {
					lang = strings.Trim(trimmed, "`")
				} else {
					lang = strings.TrimPrefix(trimmed, "[CODE START]")
				}
				lang = strings.TrimSpace(lang)
				if currentBlock != nil && lang != "" {
					currentBlock.Language = lang
				}
				inCodeBlock = true
			} else {
				// Closing
				if currentBlock != nil {
					currentBlock.Code = strings.Join(codeLines, "\n")
					blocks = append(blocks, *currentBlock)
					currentBlock = nil
				}
				inCodeBlock = false
				codeLines = []string{}
			}
			continue
		}

		// Collect code lines
		if inCodeBlock {
			codeLines = append(codeLines, line)
		}
	}

	// Handle last block
	if currentBlock != nil && len(codeLines) > 0 {
		currentBlock.Code = strings.Join(codeLines, "\n")
		blocks = append(blocks, *currentBlock)
	}

	return blocks
}

// writeFile writes a code block to the task's working directory
func (a *CoderAgent) writeFile(ctx *TaskContext, block CodeBlock) error {
	// Update task files in memory
	ctx.Files[block.FilePath] = block.Code

	// The orchestrator will write files to disk after the coder completes
	// But we should also emit an event to update the frontend
	fmt.Printf("[Coder] Generated file: %s\n", block.FilePath)

	return nil
}

// CodeExample represents a code example from MCP
type CodeExample struct {
	Source   string `json:"source"`
	Code     string `json:"code"`
	Language string `json:"language"`
}

// InferLanguage detects language from file extension
func InferLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	langMap := map[string]string{
		".js":         "javascript",
		".ts":         "typescript",
		".py":         "python",
		".go":         "go",
		".java":       "java",
		".rs":         "rust",
		".cpp":        "cpp",
		".c":          "c",
		".h":          "c",
		".hpp":        "cpp",
		".rb":         "ruby",
		".php":        "php",
		".swift":      "swift",
		".kt":         "kotlin",
		".scala":      "scala",
		".r":          "r",
		".m":          "objective-c",
		".cs":         "csharp",
		".fs":         "fsharp",
		".ex":         "elixir",
		".exs":        "elixir",
		".erl":        "erlang",
		".hs":         "haskell",
		".lua":        "lua",
		".pl":         "perl",
		".sh":         "bash",
		".ps1":        "powershell",
		".sql":        "sql",
		".html":       "html",
		".css":        "css",
		".scss":       "scss",
		".sass":       "sass",
		".less":       "less",
		".vue":        "vue",
		".json":       "json",
		".xml":        "xml",
		".yaml":       "yaml",
		".yml":        "yaml",
		".toml":       "toml",
		".md":         "markdown",
		".dockerfile": "dockerfile",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return "plaintext"
}

// ExtractImports extracts import statements from code
func ExtractImports(code string, language string) []string {
	var imports []string
	lines := strings.Split(code, "\n")

	switch language {
	case "python":
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
				imports = append(imports, trimmed)
			}
		}
	case "javascript", "typescript":
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "require(") {
				imports = append(imports, trimmed)
			}
		}
	case "go":
		inImport := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import (") {
				inImport = true
				continue
			}
			if inImport && trimmed == ")" {
				inImport = false
				continue
			}
			if inImport || strings.HasPrefix(trimmed, "import ") {
				imports = append(imports, trimmed)
			}
		}
	}

	return imports
}

// ValidateCode performs basic validation on generated code
func ValidateCode(code string, language string) []string {
	var issues []string

	// Check for common issues
	if strings.TrimSpace(code) == "" {
		issues = append(issues, "Code is empty")
	}

	// Language-specific checks
	switch language {
	case "python":
		if strings.Contains(code, "print ") && !strings.Contains(code, "print(") {
			issues = append(issues, "Using Python 2 print syntax")
		}
	case "javascript", "typescript":
		openBraces := strings.Count(code, "{")
		closeBraces := strings.Count(code, "}")
		if openBraces != closeBraces {
			issues = append(issues, "Mismatched braces")
		}
	}

	return issues
}

// FormatJSON formats JSON string with indentation
func FormatJSON(jsonStr string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", err
	}
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}
