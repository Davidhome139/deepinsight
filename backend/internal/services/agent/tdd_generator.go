package agent

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// TestDrivenGenerator implements test-driven development workflow
type TestDrivenGenerator struct {
	baseAgent
	testRunner *TestRunner
}

// TestRunner runs tests and collects results
type TestRunner struct {
	workDir  string
	language string
}

// TestResult represents a test execution result
type TestResult struct {
	Name       string        `json:"name"`
	Status     string        `json:"status"` // passed, failed, skipped, error
	Duration   time.Duration `json:"duration"`
	Message    string        `json:"message,omitempty"`
	StackTrace string        `json:"stack_trace,omitempty"`
	Coverage   float64       `json:"coverage,omitempty"`
}

// TestSuite represents a collection of test results
type TestSuite struct {
	Name       string        `json:"name"`
	Tests      []TestResult  `json:"tests"`
	TotalTests int           `json:"total_tests"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Skipped    int           `json:"skipped"`
	Duration   time.Duration `json:"duration"`
	Coverage   float64       `json:"coverage"`
}

// TDDPlan represents a test-driven development plan
type TDDPlan struct {
	Feature        string     `json:"feature"`
	TestCases      []TestCase `json:"test_cases"`
	Implementation string     `json:"implementation"`
	RefactorNotes  []string   `json:"refactor_notes,omitempty"`
}

// TestCase represents a single test case
type TestCase struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Input       string `json:"input"`
	Expected    string `json:"expected"`
	Category    string `json:"category"` // unit, integration, e2e
}

// NewTestDrivenGenerator creates a new TDD generator
func NewTestDrivenGenerator() *TestDrivenGenerator {
	return &TestDrivenGenerator{
		baseAgent: baseAgent{
			name:        "tdd-generator",
			role:        "test-driven-developer",
			description: "Generates code following Test-Driven Development principles",
			prompt: `You are a Test-Driven Development (TDD) expert. You follow the RED-GREEN-REFACTOR cycle strictly.

TDD WORKFLOW:
1. RED: Write failing tests first that describe the expected behavior
2. GREEN: Write minimal code to make the tests pass
3. REFACTOR: Improve code quality while keeping tests green

PRINCIPLES:
- Write tests BEFORE implementation
- Each test should test ONE thing
- Tests should be independent and isolated
- Follow the Arrange-Act-Assert pattern
- Aim for high test coverage (>80%)

OUTPUT FORMAT for test generation:
### Test File: [filename]_test.[ext]
` + "```" + `[language]
// Test code here
` + "```" + `

OUTPUT FORMAT for implementation:
### Implementation File: [filename].[ext]
` + "```" + `[language]
// Implementation code here
` + "```" + `

### Test Results:
- [PASS/FAIL] Test name: description

### Refactor Notes:
- Improvement suggestions`,
		},
	}
}

// Execute runs the TDD workflow
func (t *TestDrivenGenerator) Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error) {
	// Step 1: Generate test plan
	testPlan, err := t.generateTestPlan(input)
	if err != nil {
		return "", fmt.Errorf("failed to generate test plan: %v", err)
	}

	var result strings.Builder
	result.WriteString("## TDD Workflow Results\n\n")

	// Step 2: Generate tests (RED phase)
	result.WriteString("### Phase 1: RED - Writing Tests\n\n")
	tests, err := t.generateTests(testPlan)
	if err != nil {
		return "", fmt.Errorf("failed to generate tests: %v", err)
	}
	result.WriteString(tests)
	result.WriteString("\n\n")

	// Step 3: Generate implementation (GREEN phase)
	result.WriteString("### Phase 2: GREEN - Implementation\n\n")
	impl, err := t.generateImplementation(testPlan, tests)
	if err != nil {
		return "", fmt.Errorf("failed to generate implementation: %v", err)
	}
	result.WriteString(impl)
	result.WriteString("\n\n")

	// Step 4: Refactor suggestions
	result.WriteString("### Phase 3: REFACTOR - Suggestions\n\n")
	refactor, err := t.generateRefactorSuggestions(tests, impl)
	if err == nil {
		result.WriteString(refactor)
	}

	return result.String(), nil
}

// generateTestPlan generates a test plan for the feature
func (t *TestDrivenGenerator) generateTestPlan(feature string) (*TDDPlan, error) {
	prompt := fmt.Sprintf(`Create a TDD plan for the following feature:
%s

Generate a comprehensive test plan with:
1. Feature name
2. Test cases (name, description, input, expected output, category)
3. Key implementation notes

Output in JSON format:
{
  "feature": "name",
  "test_cases": [
    {"name": "", "description": "", "input": "", "expected": "", "category": "unit"}
  ],
  "implementation": "notes"
}`, feature)

	response, err := t.CallLLM(prompt)
	if err != nil {
		return nil, err
	}

	// Parse the plan (simplified - in production use proper JSON extraction)
	plan := &TDDPlan{
		Feature: feature,
		TestCases: []TestCase{
			{Name: "test_basic_functionality", Description: "Tests basic feature", Category: "unit"},
			{Name: "test_edge_cases", Description: "Tests edge cases", Category: "unit"},
			{Name: "test_error_handling", Description: "Tests error scenarios", Category: "unit"},
		},
	}

	// Try to parse actual JSON from response
	if jsonStart := strings.Index(response, "{"); jsonStart >= 0 {
		if jsonEnd := strings.LastIndex(response, "}"); jsonEnd > jsonStart {
			plan.Implementation = response[jsonStart : jsonEnd+1]
		}
	}

	return plan, nil
}

// generateTests generates test code based on the plan
func (t *TestDrivenGenerator) generateTests(plan *TDDPlan) (string, error) {
	prompt := fmt.Sprintf(`Generate comprehensive test code for:
Feature: %s

Test cases to implement:
%v

Generate tests that:
1. Are independent and isolated
2. Follow Arrange-Act-Assert pattern
3. Have descriptive names
4. Cover edge cases and error scenarios
5. Include setup and teardown if needed

Output the complete test file with all tests.`, plan.Feature, plan.TestCases)

	return t.CallLLM(prompt)
}

// generateImplementation generates implementation to pass tests
func (t *TestDrivenGenerator) generateImplementation(plan *TDDPlan, tests string) (string, error) {
	prompt := fmt.Sprintf(`Generate minimal implementation to make these tests pass:

Feature: %s

Tests:
%s

Requirements:
1. Write ONLY enough code to pass the tests
2. Follow best practices and coding standards
3. Include proper error handling
4. Add necessary imports and dependencies
5. Keep it simple - don't over-engineer

Output the complete implementation file.`, plan.Feature, tests)

	return t.CallLLM(prompt)
}

// generateRefactorSuggestions generates refactoring suggestions
func (t *TestDrivenGenerator) generateRefactorSuggestions(tests, impl string) (string, error) {
	prompt := fmt.Sprintf(`Review this TDD implementation and suggest refactoring improvements:

Tests:
%s

Implementation:
%s

Suggest improvements for:
1. Code readability
2. Performance optimizations
3. Design patterns that could be applied
4. Better error handling
5. Documentation improvements

Keep tests passing after refactoring!`, tests, impl)

	return t.CallLLM(prompt)
}

// NewTestRunner creates a new test runner
func NewTestRunner(workDir, language string) *TestRunner {
	return &TestRunner{
		workDir:  workDir,
		language: language,
	}
}

// RunTests runs all tests and returns results
func (r *TestRunner) RunTests() (*TestSuite, error) {
	suite := &TestSuite{
		Name:  "AI Generated Tests",
		Tests: make([]TestResult, 0),
	}

	var cmd *exec.Cmd
	var testPattern *regexp.Regexp

	// Language-specific test commands
	switch r.language {
	case "go", "golang":
		cmd = exec.Command("go", "test", "-v", "-json", "./...")
		cmd.Dir = r.workDir
		testPattern = regexp.MustCompile(`(?m)^=== RUN\s+(\S+)$|^--- (PASS|FAIL|SKIP):\s+(\S+)\s+\(([0-9.]+)s\)$`)
	case "python":
		cmd = exec.Command("python", "-m", "pytest", "-v", "--tb=short")
		cmd.Dir = r.workDir
		testPattern = regexp.MustCompile(`(?m)^(\S+::\S+)\s+(PASSED|FAILED|SKIPPED)`)
	case "javascript", "typescript":
		cmd = exec.Command("npm", "test", "--", "--reporter=spec")
		cmd.Dir = r.workDir
		testPattern = regexp.MustCompile(`(?m)^\s*(✓|✗|○)\s+(.+)$`)
	case "java":
		cmd = exec.Command("mvn", "test")
		cmd.Dir = r.workDir
		testPattern = regexp.MustCompile(`(?m)^\[INFO\]\s+(Tests run: \d+, Failures: \d+, Errors: \d+, Skipped: \d+)`)
	default:
		return suite, fmt.Errorf("unsupported language for testing: %s", r.language)
	}

	// Execute tests
	startTime := time.Now()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	// Parse output based on language
	output := stdout.String() + stderr.String()

	if testPattern != nil {
		matches := testPattern.FindAllStringSubmatch(output, -1)
		for _, match := range matches {
			result := TestResult{
				Duration: duration / time.Duration(len(matches)+1),
			}

			switch r.language {
			case "go", "golang":
				if len(match) > 3 && match[2] != "" {
					result.Name = match[3]
					switch match[2] {
					case "PASS":
						result.Status = "passed"
					case "FAIL":
						result.Status = "failed"
					case "SKIP":
						result.Status = "skipped"
					}
				}
			case "python":
				if len(match) > 2 {
					result.Name = match[1]
					switch match[2] {
					case "PASSED":
						result.Status = "passed"
					case "FAILED":
						result.Status = "failed"
					case "SKIPPED":
						result.Status = "skipped"
					}
				}
			case "javascript", "typescript":
				if len(match) > 2 {
					result.Name = strings.TrimSpace(match[2])
					switch match[1] {
					case "✓":
						result.Status = "passed"
					case "✗":
						result.Status = "failed"
					case "○":
						result.Status = "skipped"
					}
				}
			}

			if result.Name != "" && result.Status != "" {
				suite.Tests = append(suite.Tests, result)
			}
		}
	}

	// If no tests parsed, create summary result
	if len(suite.Tests) == 0 {
		status := "passed"
		if err != nil {
			status = "failed"
		}
		suite.Tests = append(suite.Tests, TestResult{
			Name:     "test_execution",
			Status:   status,
			Duration: duration,
			Message:  output,
		})
	}

	// Calculate summary
	for _, test := range suite.Tests {
		suite.Duration += test.Duration
		switch test.Status {
		case "passed":
			suite.Passed++
		case "failed":
			suite.Failed++
		case "skipped":
			suite.Skipped++
		}
	}
	suite.TotalTests = len(suite.Tests)

	return suite, err
}

// GetTestCommand returns the appropriate test command for the language
func (r *TestRunner) GetTestCommand() string {
	switch r.language {
	case "go", "golang":
		return "go test -v ./..."
	case "python":
		return "python -m pytest -v"
	case "javascript", "typescript":
		return "npm test"
	case "java":
		return "mvn test"
	default:
		return ""
	}
}

// IsTestEnvironmentReady checks if the test environment is ready
func (r *TestRunner) IsTestEnvironmentReady() bool {
	var cmd *exec.Cmd
	switch r.language {
	case "go", "golang":
		cmd = exec.Command("go", "version")
	case "python":
		cmd = exec.Command("python", "--version")
	case "javascript", "typescript":
		cmd = exec.Command("npm", "--version")
	case "java":
		cmd = exec.Command("java", "-version")
	default:
		return false
	}
	return cmd.Run() == nil
}

// GenerateTestFile generates a test file from test cases
func GenerateTestFile(language string, testCases []TestCase, targetFile string) string {
	var sb strings.Builder

	switch language {
	case "go":
		sb.WriteString("package main\n\n")
		sb.WriteString("import (\n\t\"testing\"\n)\n\n")
		for _, tc := range testCases {
			sb.WriteString(fmt.Sprintf("func Test%s(t *testing.T) {\n", toPascalCase(tc.Name)))
			sb.WriteString(fmt.Sprintf("\t// %s\n", tc.Description))
			sb.WriteString("\t// Arrange\n\n")
			sb.WriteString("\t// Act\n\n")
			sb.WriteString("\t// Assert\n")
			sb.WriteString("\t// TODO: Implement test\n")
			sb.WriteString("}\n\n")
		}

	case "python":
		sb.WriteString("import unittest\n\n")
		sb.WriteString("class TestGenerated(unittest.TestCase):\n")
		for _, tc := range testCases {
			sb.WriteString(fmt.Sprintf("\n    def %s(self):\n", tc.Name))
			sb.WriteString(fmt.Sprintf("        \"\"\"%s\"\"\"\n", tc.Description))
			sb.WriteString("        # Arrange\n\n")
			sb.WriteString("        # Act\n\n")
			sb.WriteString("        # Assert\n")
			sb.WriteString("        pass  # TODO: Implement test\n")
		}
		sb.WriteString("\n\nif __name__ == '__main__':\n")
		sb.WriteString("    unittest.main()\n")

	case "javascript", "typescript":
		sb.WriteString("describe('Generated Tests', () => {\n")
		for _, tc := range testCases {
			sb.WriteString(fmt.Sprintf("  it('%s', () => {\n", tc.Description))
			sb.WriteString("    // Arrange\n\n")
			sb.WriteString("    // Act\n\n")
			sb.WriteString("    // Assert\n")
			sb.WriteString("    // TODO: Implement test\n")
			sb.WriteString("  });\n\n")
		}
		sb.WriteString("});\n")

	case "java":
		sb.WriteString("import org.junit.jupiter.api.Test;\n")
		sb.WriteString("import static org.junit.jupiter.api.Assertions.*;\n\n")
		sb.WriteString("public class GeneratedTests {\n\n")
		for _, tc := range testCases {
			sb.WriteString("    @Test\n")
			sb.WriteString(fmt.Sprintf("    void %s() {\n", tc.Name))
			sb.WriteString(fmt.Sprintf("        // %s\n", tc.Description))
			sb.WriteString("        // Arrange\n\n")
			sb.WriteString("        // Act\n\n")
			sb.WriteString("        // Assert\n")
			sb.WriteString("        // TODO: Implement test\n")
			sb.WriteString("    }\n\n")
		}
		sb.WriteString("}\n")
	}

	return sb.String()
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, "")
}

// ParseTestResults parses test output into structured results
func ParseTestResults(language, output string) *TestSuite {
	suite := &TestSuite{
		Tests: make([]TestResult, 0),
	}

	lines := strings.Split(output, "\n")

	switch language {
	case "go":
		for _, line := range lines {
			if strings.Contains(line, "--- PASS:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					suite.Tests = append(suite.Tests, TestResult{
						Name:   parts[2],
						Status: "passed",
					})
					suite.Passed++
				}
			} else if strings.Contains(line, "--- FAIL:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					suite.Tests = append(suite.Tests, TestResult{
						Name:   parts[2],
						Status: "failed",
					})
					suite.Failed++
				}
			}
		}

	case "python":
		for _, line := range lines {
			if strings.HasPrefix(line, "ok") || strings.Contains(line, "OK") {
				suite.Passed++
			} else if strings.HasPrefix(line, "FAIL") || strings.Contains(line, "FAILED") {
				suite.Failed++
			}
		}

	case "javascript", "typescript":
		for _, line := range lines {
			if strings.Contains(line, "✓") || strings.Contains(line, "passing") {
				suite.Passed++
			} else if strings.Contains(line, "✗") || strings.Contains(line, "failing") {
				suite.Failed++
			}
		}
	}

	suite.TotalTests = suite.Passed + suite.Failed + suite.Skipped
	return suite
}
