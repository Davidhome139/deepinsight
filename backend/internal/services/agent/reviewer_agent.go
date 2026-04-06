package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ReviewerAgent reviews code for quality and best practices
type ReviewerAgent struct {
	baseAgent
}

// CodeReview represents a code review result
type CodeReview struct {
	FilePath string      `json:"file_path"`
	Issues   []CodeIssue `json:"issues"`
	Score    int         `json:"score"`
	Summary  string      `json:"summary"`
}

// CodeIssue represents a single code issue
type CodeIssue struct {
	Severity   string `json:"severity"`
	Line       int    `json:"line"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	Category   string `json:"category"`
}

// NewReviewerAgent creates a new reviewer agent
func NewReviewerAgent() *ReviewerAgent {
	agent := &ReviewerAgent{}

	// Try to load configuration from agent_configs.json
	configPath := "plans/agent_configs.json"
	configData, err := os.ReadFile(configPath)
	if err == nil {
		// Parse JSON data
		var configs []AgentConfig
		if err := json.Unmarshal(configData, &configs); err == nil {
			// Find the reviewer agent configuration
			for _, config := range configs {
				if config.Name == "reviewer" {
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
	agent.name = "reviewer"
	agent.role = "reviewer"
	agent.description = "Reviews code for quality and best practices"
	agent.prompt = `You are a senior code reviewer. Your job is to review code for quality, security, and best practices.

When reviewing code, check for:
1. Security vulnerabilities (SQL injection, XSS, etc.)
2. Performance issues (inefficient algorithms, memory leaks)
3. Code style violations
4. Error handling
5. Documentation
6. Test coverage
7. Design patterns and architecture

Format your review as:

SUMMARY: <brief overall assessment>

SCORE: <0-100>

ISSUES:
- [SEVERITY] Line X: <issue description>
  Suggestion: <how to fix>

POSITIVE:
- <what's done well>

RECOMMENDATIONS:
- <general improvements>

Be thorough but constructive.`

	return agent
}

// Execute runs the reviewer agent
func (a *ReviewerAgent) Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error) {
	// Determine what to review
	filePath, content := a.determineReviewTarget(ctx, input)

	if content == "" {
		return "", fmt.Errorf("no code found to review")
	}

	// Enhance input with code
	enhancedInput := a.enhanceInput(filePath, content)

	// Call LLM to review
	response, err := a.callLLM(enhancedInput)
	if err != nil {
		return "", err
	}

	// Parse review
	review := a.parseReview(response, filePath)

	// Format output
	return a.formatReview(review), nil
}

// determineReviewTarget figures out what file to review
func (a *ReviewerAgent) determineReviewTarget(ctx *TaskContext, input string) (string, string) {
	// Check if input specifies a file
	input = strings.TrimSpace(input)

	// Try to extract file path from input
	if strings.HasPrefix(input, "review ") {
		filePath := strings.TrimPrefix(input, "review ")
		filePath = strings.TrimSpace(filePath)

		if content, ok := ctx.Files[filePath]; ok {
			return filePath, content
		}
	}

	// Review the most recently modified file
	var latestFile string
	var latestContent string

	for path, content := range ctx.Files {
		if latestFile == "" || len(content) > len(latestContent) {
			latestFile = path
			latestContent = content
		}
	}

	return latestFile, latestContent
}

// enhanceInput prepares the review input
func (a *ReviewerAgent) enhanceInput(filePath string, content string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("File: %s\n\n", filePath))
	sb.WriteString("Code:\n```\n")

	// Add line numbers
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		sb.WriteString(fmt.Sprintf("%4d | %s\n", i+1, line))
	}

	sb.WriteString("```\n\n")
	sb.WriteString(a.prompt)

	return sb.String()
}

// callLLM calls the language model
func (a *ReviewerAgent) callLLM(input string) (string, error) {
	// Use the base agent's CallLLM method
	return a.CallLLM(input)
}

// parseReview extracts review information from response
func (a *ReviewerAgent) parseReview(response string, filePath string) *CodeReview {
	review := &CodeReview{
		FilePath: filePath,
		Issues:   []CodeIssue{},
	}

	lines := strings.Split(response, "\n")
	var currentSection string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Parse sections
		if strings.HasPrefix(trimmed, "SUMMARY:") {
			currentSection = "summary"
			review.Summary = strings.TrimPrefix(trimmed, "SUMMARY:")
			continue
		}

		if strings.HasPrefix(trimmed, "SCORE:") {
			scoreStr := strings.TrimPrefix(trimmed, "SCORE:")
			fmt.Sscanf(scoreStr, "%d", &review.Score)
			continue
		}

		if strings.HasPrefix(trimmed, "ISSUES:") {
			currentSection = "issues"
			continue
		}

		if strings.HasPrefix(trimmed, "POSITIVE:") {
			currentSection = "positive"
			continue
		}

		if strings.HasPrefix(trimmed, "RECOMMENDATIONS:") {
			currentSection = "recommendations"
			continue
		}

		// Parse issues
		if currentSection == "issues" && strings.HasPrefix(trimmed, "-") {
			issue := a.parseIssueLine(trimmed)
			if issue != nil {
				review.Issues = append(review.Issues, *issue)
			}
		}
	}

	return review
}

// parseIssueLine parses an issue line
func (a *ReviewerAgent) parseIssueLine(line string) *CodeIssue {
	issue := &CodeIssue{}

	// Extract severity
	if strings.Contains(line, "[CRITICAL]") {
		issue.Severity = "critical"
	} else if strings.Contains(line, "[HIGH]") {
		issue.Severity = "high"
	} else if strings.Contains(line, "[MEDIUM]") {
		issue.Severity = "medium"
	} else if strings.Contains(line, "[LOW]") {
		issue.Severity = "low"
	}

	// Extract line number
	if idx := strings.Index(line, "Line "); idx >= 0 {
		fmt.Sscanf(line[idx:], "Line %d", &issue.Line)
	}

	// Extract message (after the severity/line info)
	parts := strings.SplitN(line, ":", 2)
	if len(parts) > 1 {
		issue.Message = strings.TrimSpace(parts[1])
	}

	return issue
}

// formatReview formats the review for output
func (a *ReviewerAgent) formatReview(review *CodeReview) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Code Review: %s\n", review.FilePath))
	sb.WriteString(fmt.Sprintf("Score: %d/100\n\n", review.Score))

	if review.Summary != "" {
		sb.WriteString(fmt.Sprintf("Summary: %s\n\n", review.Summary))
	}

	if len(review.Issues) > 0 {
		sb.WriteString(fmt.Sprintf("Found %d issues:\n\n", len(review.Issues)))

		// Group by severity
		severityOrder := []string{"critical", "high", "medium", "low"}
		for _, sev := range severityOrder {
			for _, issue := range review.Issues {
				if issue.Severity == sev {
					sb.WriteString(fmt.Sprintf("[%s] Line %d: %s\n",
						strings.ToUpper(sev), issue.Line, issue.Message))
					if issue.Suggestion != "" {
						sb.WriteString(fmt.Sprintf("  Suggestion: %s\n", issue.Suggestion))
					}
					sb.WriteString("\n")
				}
			}
		}
	} else {
		sb.WriteString("No issues found!\n")
	}

	return sb.String()
}

// CalculateScore calculates a code quality score
func (a *ReviewerAgent) CalculateScore(issues []CodeIssue) int {
	baseScore := 100

	// Deduct points based on severity
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			baseScore -= 20
		case "high":
			baseScore -= 10
		case "medium":
			baseScore -= 5
		case "low":
			baseScore -= 2
		}
	}

	if baseScore < 0 {
		baseScore = 0
	}
	return baseScore
}

// SecurityCheck performs basic security checks
func (a *ReviewerAgent) SecurityCheck(code string, language string) []CodeIssue {
	var issues []CodeIssue

	// Language-specific security checks
	switch language {
	case "python":
		issues = append(issues, a.checkPythonSecurity(code)...)
	case "javascript", "typescript":
		issues = append(issues, a.checkJavaScriptSecurity(code)...)
	case "go":
		issues = append(issues, a.checkGoSecurity(code)...)
	case "java":
		issues = append(issues, a.checkJavaSecurity(code)...)
	}

	// Common checks for all languages
	issues = append(issues, a.checkCommonSecurity(code)...)

	return issues
}

// checkPythonSecurity checks for Python security issues
func (a *ReviewerAgent) checkPythonSecurity(code string) []CodeIssue {
	var issues []CodeIssue
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// SQL injection
		if strings.Contains(trimmed, "execute(") && strings.Contains(trimmed, "%") {
			issues = append(issues, CodeIssue{
				Severity: "critical",
				Line:     i + 1,
				Message:  "Potential SQL injection vulnerability",
				Category: "security",
			})
		}

		// eval/exec
		if strings.Contains(trimmed, "eval(") || strings.Contains(trimmed, "exec(") {
			issues = append(issues, CodeIssue{
				Severity: "high",
				Line:     i + 1,
				Message:  "Use of eval/exec can be dangerous",
				Category: "security",
			})
		}

		// Hardcoded secrets
		if (strings.Contains(trimmed, "password") || strings.Contains(trimmed, "secret") ||
			strings.Contains(trimmed, "api_key") || strings.Contains(trimmed, "token")) &&
			strings.Contains(trimmed, "=") {
			issues = append(issues, CodeIssue{
				Severity: "high",
				Line:     i + 1,
				Message:  "Possible hardcoded secret",
				Category: "security",
			})
		}
	}

	return issues
}

// checkJavaScriptSecurity checks for JavaScript security issues
func (a *ReviewerAgent) checkJavaScriptSecurity(code string) []CodeIssue {
	var issues []CodeIssue
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// innerHTML XSS
		if strings.Contains(trimmed, "innerHTML") {
			issues = append(issues, CodeIssue{
				Severity: "high",
				Line:     i + 1,
				Message:  "innerHTML can lead to XSS vulnerabilities",
				Category: "security",
			})
		}

		// eval
		if strings.Contains(trimmed, "eval(") {
			issues = append(issues, CodeIssue{
				Severity: "critical",
				Line:     i + 1,
				Message:  "eval() is dangerous and should be avoided",
				Category: "security",
			})
		}

		// document.write
		if strings.Contains(trimmed, "document.write") {
			issues = append(issues, CodeIssue{
				Severity: "medium",
				Line:     i + 1,
				Message:  "document.write is deprecated and can cause issues",
				Category: "best_practice",
			})
		}
	}

	return issues
}

// checkGoSecurity checks for Go security issues
func (a *ReviewerAgent) checkGoSecurity(code string) []CodeIssue {
	var issues []CodeIssue
	// Add Go-specific checks
	return issues
}

// checkJavaSecurity checks for Java security issues
func (a *ReviewerAgent) checkJavaSecurity(code string) []CodeIssue {
	var issues []CodeIssue
	// Add Java-specific checks
	return issues
}

// checkCommonSecurity checks for common security issues
func (a *ReviewerAgent) checkCommonSecurity(code string) []CodeIssue {
	var issues []CodeIssue
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		trimmed := strings.ToLower(strings.TrimSpace(line))

		// TODO/FIXME comments
		if strings.Contains(trimmed, "todo") || strings.Contains(trimmed, "fixme") {
			issues = append(issues, CodeIssue{
				Severity: "low",
				Line:     i + 1,
				Message:  "TODO/FIXME comment found",
				Category: "maintenance",
			})
		}

		// Console/log statements
		if strings.Contains(trimmed, "console.log") || strings.Contains(trimmed, "print(") {
			issues = append(issues, CodeIssue{
				Severity: "low",
				Line:     i + 1,
				Message:  "Debug print statement should be removed",
				Category: "cleanup",
			})
		}
	}

	return issues
}

// ReviewCategory represents review categories
type ReviewCategory string

const (
	CategorySecurity        ReviewCategory = "security"
	CategoryPerformance     ReviewCategory = "performance"
	CategoryStyle           ReviewCategory = "style"
	CategoryMaintainability ReviewCategory = "maintainability"
	CategoryDocumentation   ReviewCategory = "documentation"
)

// ReviewCriteria defines review criteria
type ReviewCriteria struct {
	Category    ReviewCategory
	Weight      float64
	Description string
}

// DefaultReviewCriteria returns default review criteria
func DefaultReviewCriteria() []ReviewCriteria {
	return []ReviewCriteria{
		{CategorySecurity, 0.30, "Security vulnerabilities"},
		{CategoryPerformance, 0.20, "Performance issues"},
		{CategoryMaintainability, 0.25, "Code maintainability"},
		{CategoryStyle, 0.15, "Code style"},
		{CategoryDocumentation, 0.10, "Documentation"},
	}
}
