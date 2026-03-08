package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// DebuggerAgent analyzes and fixes errors
type DebuggerAgent struct {
	baseAgent
}

// ErrorAnalysis represents the analysis of an error
type ErrorAnalysis struct {
	ErrorType    string `json:"error_type"`
	RootCause    string `json:"root_cause"`
	SuggestedFix string `json:"suggested_fix"`
	Prevention   string `json:"prevention"`
}

// NewDebuggerAgent creates a new debugger agent
func NewDebuggerAgent() *DebuggerAgent {
	agent := &DebuggerAgent{}

	// Try to load configuration from agent_configs.json
	configPath := "plans/agent_configs.json"
	configData, err := os.ReadFile(configPath)
	if err == nil {
		// Parse JSON data
		var configs []AgentConfig
		if err := json.Unmarshal(configData, &configs); err == nil {
			// Find the debugger agent configuration
			for _, config := range configs {
				if config.Name == "debugger" {
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
	agent.name = "debugger"
	agent.role = "debugger"
	agent.description = "Analyzes errors and suggests fixes"
	agent.prompt = `You are an expert debugger. Your job is to analyze errors and provide fixes.

When analyzing an error:
1. Identify the error type (syntax, runtime, logic, etc.)
2. Find the root cause
3. Provide a specific fix with code changes
4. Suggest prevention strategies

Format your response:

ANALYSIS: <brief analysis of the error>

ROOT CAUSE: <detailed explanation of why the error occurred>

FIX: <specific code changes needed>

PREVENTION: <how to avoid this in the future>

Be specific and actionable. Provide actual code that can be used.`

	return agent
}

// Execute runs the debugger agent
func (a *DebuggerAgent) Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error) {
	// Parse error from input
	errorInfo := a.parseError(input)

	// Enhance input with context
	enhancedInput := a.enhanceInput(ctx, errorInfo)

	// Search for similar errors via MCP
	solutions, err := a.searchSolutions(mcpManager, errorInfo)
	if err == nil && len(solutions) > 0 {
		enhancedInput = a.addSolutionsToInput(enhancedInput, solutions)
	}

	// Call LLM to analyze error
	response, err := a.callLLM(enhancedInput)
	if err != nil {
		return "", err
	}

	// Extract and validate fix
	analysis := a.extractAnalysis(response)
	if analysis == nil {
		return response, nil // Return raw response if parsing fails
	}

	return a.formatFix(analysis), nil
}

// ErrorInfo represents parsed error information
type ErrorInfo struct {
	RawError   string
	ErrorType  string
	Message    string
	FilePath   string
	LineNumber int
	StackTrace string
	Context    string
}

// parseError extracts error information from input
func (a *DebuggerAgent) parseError(input string) *ErrorInfo {
	info := &ErrorInfo{
		RawError: input,
	}

	// Try to extract file path and line number
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		// Common patterns
		if strings.Contains(line, "File \"") && strings.Contains(line, "\", line") {
			// Python traceback pattern
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				info.FilePath = parts[1]
			}
		}

		if strings.Contains(line, ".go:") {
			// Go error pattern
			idx := strings.Index(line, ".go:")
			if idx > 0 {
				start := strings.LastIndex(line[:idx], " ")
				if start < 0 {
					start = 0
				}
				info.FilePath = strings.TrimSpace(line[start : idx+3])
			}
		}
	}

	// Detect error type
	lower := strings.ToLower(input)
	switch {
	case strings.Contains(lower, "syntax error"):
		info.ErrorType = "syntax"
	case strings.Contains(lower, "runtime error"):
		info.ErrorType = "runtime"
	case strings.Contains(lower, "type error"):
		info.ErrorType = "type"
	case strings.Contains(lower, "reference error"):
		info.ErrorType = "reference"
	case strings.Contains(lower, "undefined"):
		info.ErrorType = "undefined"
	case strings.Contains(lower, "null pointer"):
		info.ErrorType = "null_pointer"
	case strings.Contains(lower, "index out of range"):
		info.ErrorType = "index_out_of_range"
	case strings.Contains(lower, "permission denied"):
		info.ErrorType = "permission"
	case strings.Contains(lower, "not found") || strings.Contains(lower, "enoent"):
		info.ErrorType = "not_found"
	case strings.Contains(lower, "connection"):
		info.ErrorType = "connection"
	default:
		info.ErrorType = "unknown"
	}

	return info
}

// enhanceInput adds context to the input
func (a *DebuggerAgent) enhanceInput(ctx *TaskContext, errorInfo *ErrorInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Error: %s\n", errorInfo.RawError))
	sb.WriteString(fmt.Sprintf("Error Type: %s\n", errorInfo.ErrorType))

	if errorInfo.FilePath != "" {
		sb.WriteString(fmt.Sprintf("File: %s\n", errorInfo.FilePath))

		// Include file content if available
		if content, ok := ctx.Files[errorInfo.FilePath]; ok {
			sb.WriteString("\nFile content:\n")
			if len(content) > 2000 {
				sb.WriteString(content[:2000])
				sb.WriteString("\n... (truncated)\n")
			} else {
				sb.WriteString(content)
			}
		}
	}

	if len(ctx.AgentHistory) > 0 {
		sb.WriteString("\nRecent agent actions:\n")
		start := len(ctx.AgentHistory) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(ctx.AgentHistory); i++ {
			exec := ctx.AgentHistory[i]
			sb.WriteString(fmt.Sprintf("- %s: %s\n", exec.AgentName, exec.Input[:min(len(exec.Input), 100)]))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(a.prompt)

	return sb.String()
}

// searchSolutions searches for similar error solutions
func (a *DebuggerAgent) searchSolutions(mcpManager *MCPManager, errorInfo *ErrorInfo) ([]string, error) {
	if mcpManager == nil {
		return nil, fmt.Errorf("MCP manager not available")
	}

	query := fmt.Sprintf("%s error fix solution", errorInfo.ErrorType)
	if errorInfo.Message != "" {
		query = errorInfo.Message + " fix"
	}

	return mcpManager.SearchErrorSolutions(query)
}

// addSolutionsToInput adds found solutions to input
func (a *DebuggerAgent) addSolutionsToInput(input string, solutions []string) string {
	var sb strings.Builder
	sb.WriteString(input)
	sb.WriteString("\n\nSimilar error solutions found:\n")

	for i, solution := range solutions {
		sb.WriteString(fmt.Sprintf("\nSolution %d:\n%s\n", i+1, solution))
	}

	return sb.String()
}

// callLLM calls the language model
func (a *DebuggerAgent) callLLM(input string) (string, error) {
	// Use the base agent's CallLLM method
	return a.CallLLM(input)
}

// extractAnalysis parses the LLM response
func (a *DebuggerAgent) extractAnalysis(response string) *ErrorAnalysis {
	analysis := &ErrorAnalysis{}

	lines := strings.Split(response, "\n")
	var currentSection string
	var sectionContent []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section headers
		if strings.HasPrefix(trimmed, "ANALYSIS:") {
			if currentSection != "" && len(sectionContent) > 0 {
				a.setAnalysisField(analysis, currentSection, strings.Join(sectionContent, "\n"))
			}
			currentSection = "analysis"
			sectionContent = []string{strings.TrimPrefix(trimmed, "ANALYSIS:")}
			continue
		}

		if strings.HasPrefix(trimmed, "ROOT CAUSE:") {
			if currentSection != "" && len(sectionContent) > 0 {
				a.setAnalysisField(analysis, currentSection, strings.Join(sectionContent, "\n"))
			}
			currentSection = "root_cause"
			sectionContent = []string{strings.TrimPrefix(trimmed, "ROOT CAUSE:")}
			continue
		}

		if strings.HasPrefix(trimmed, "FIX:") {
			if currentSection != "" && len(sectionContent) > 0 {
				a.setAnalysisField(analysis, currentSection, strings.Join(sectionContent, "\n"))
			}
			currentSection = "fix"
			sectionContent = []string{strings.TrimPrefix(trimmed, "FIX:")}
			continue
		}

		if strings.HasPrefix(trimmed, "PREVENTION:") {
			if currentSection != "" && len(sectionContent) > 0 {
				a.setAnalysisField(analysis, currentSection, strings.Join(sectionContent, "\n"))
			}
			currentSection = "prevention"
			sectionContent = []string{strings.TrimPrefix(trimmed, "PREVENTION:")}
			continue
		}

		// Collect content
		if currentSection != "" {
			sectionContent = append(sectionContent, line)
		}
	}

	// Save last section
	if currentSection != "" && len(sectionContent) > 0 {
		a.setAnalysisField(analysis, currentSection, strings.Join(sectionContent, "\n"))
	}

	return analysis
}

// setAnalysisField sets a field in the analysis struct
func (a *DebuggerAgent) setAnalysisField(analysis *ErrorAnalysis, field string, value string) {
	value = strings.TrimSpace(value)
	switch field {
	case "analysis":
		// Not stored separately
	case "root_cause":
		analysis.RootCause = value
	case "fix":
		analysis.SuggestedFix = value
	case "prevention":
		analysis.Prevention = value
	}
}

// formatFix formats the fix for output
func (a *DebuggerAgent) formatFix(analysis *ErrorAnalysis) string {
	var sb strings.Builder

	if analysis.RootCause != "" {
		sb.WriteString(fmt.Sprintf("Root Cause: %s\n\n", analysis.RootCause))
	}

	if analysis.SuggestedFix != "" {
		sb.WriteString(fmt.Sprintf("Suggested Fix:\n%s\n\n", analysis.SuggestedFix))
	}

	if analysis.Prevention != "" {
		sb.WriteString(fmt.Sprintf("Prevention: %s\n", analysis.Prevention))
	}

	return sb.String()
}

// CommonErrorPatterns contains patterns for common errors
var CommonErrorPatterns = map[string]string{
	"undefined variable":          "Check variable spelling and scope. Ensure the variable is declared before use.",
	"null pointer exception":      "Add null checks before accessing object properties or methods.",
	"index out of range":          "Check array/slice bounds before accessing elements. Use len() to verify size.",
	"syntax error":                "Check for missing brackets, parentheses, or semicolons.",
	"type mismatch":               "Ensure variables are of the expected type. Use type conversion if necessary.",
	"import error":                "Check that the module/package is installed and the import path is correct.",
	"permission denied":           "Check file permissions. Use chmod or run with appropriate privileges.",
	"connection refused":          "Verify the service is running and the port is correct. Check firewall settings.",
	"timeout":                     "Increase timeout duration or optimize the operation. Check network connectivity.",
	"memory leak":                 "Ensure resources are properly released. Use defer/try-finally patterns.",
	"race condition":              "Use mutexes, channels, or atomic operations for shared state access.",
	"deadlock":                    "Check lock ordering. Ensure locks are always released.",
	"stack overflow":              "Check for infinite recursion. Add base cases to recursive functions.",
	"segmentation fault":          "Check pointer validity. Ensure memory is properly allocated.",
	"divide by zero":              "Add checks to ensure divisor is not zero before division.",
	"file not found":              "Verify the file path exists. Check working directory.",
	"module not found":            "Install missing dependencies. Check package.json, requirements.txt, etc.",
	"deprecated":                  "Update to the recommended alternative. Check documentation.",
	"unhandled promise rejection": "Add .catch() or try-catch for async operations.",
	"circular dependency":         "Refactor to break the circular reference. Use interfaces or dependency injection.",
}

// MatchErrorPattern tries to match an error to a known pattern
func MatchErrorPattern(errorMsg string) (string, bool) {
	lower := strings.ToLower(errorMsg)
	for pattern, solution := range CommonErrorPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return solution, true
		}
	}
	return "", false
}

// ExtractErrorLocation extracts file and line info from stack traces
func ExtractErrorLocation(stackTrace string) (file string, line int, function string) {
	lines := strings.Split(stackTrace, "\n")

	for _, l := range lines {
		l = strings.TrimSpace(l)

		// Python traceback
		if strings.Contains(l, "File \"") && strings.Contains(l, "\", line ") {
			parts := strings.Split(l, "\"")
			if len(parts) >= 2 {
				file = parts[1]
			}
			if strings.Contains(l, "\", line ") {
				lineStr := l[strings.Index(l, "\", line ")+8:]
				lineStr = strings.TrimSpace(lineStr)
				if idx := strings.Index(lineStr, ","); idx > 0 {
					fmt.Sscanf(lineStr[:idx], "%d", &line)
				}
			}
		}

		// Go stack trace
		if strings.Contains(l, ".go:") {
			idx := strings.LastIndex(l, "/")
			if idx < 0 {
				idx = 0
			}
			parts := strings.Split(l[idx:], ":")
			if len(parts) >= 2 {
				file = parts[0]
				fmt.Sscanf(parts[1], "%d", &line)
			}
		}
	}

	return file, line, function
}

// GenerateStackTraceSummary creates a human-readable summary of a stack trace
func GenerateStackTraceSummary(stackTrace string) string {
	var sb strings.Builder
	lines := strings.Split(stackTrace, "\n")

	// Extract the error message (usually first line)
	if len(lines) > 0 {
		sb.WriteString(fmt.Sprintf("Error: %s\n", strings.TrimSpace(lines[0])))
	}

	// Extract relevant frames
	var frames []string
	for i, line := range lines {
		if strings.Contains(line, "File \"") || strings.Contains(line, ".go:") {
			// Get the function name (usually next line for Python)
			function := ""
			if i+1 < len(lines) {
				function = strings.TrimSpace(lines[i+1])
			}
			frames = append(frames, fmt.Sprintf("  at %s (%s)", function, strings.TrimSpace(line)))
		}
	}

	if len(frames) > 0 {
		sb.WriteString("Stack trace:\n")
		// Show last 5 frames (most relevant)
		start := len(frames) - 5
		if start < 0 {
			start = 0
		}
		for _, frame := range frames[start:] {
			sb.WriteString(frame + "\n")
		}
	}

	return sb.String()
}
