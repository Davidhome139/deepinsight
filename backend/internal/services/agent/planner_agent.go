package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PlannerAgent creates execution plans for tasks
type PlannerAgent struct {
	baseAgent
}

// NewPlannerAgent creates a new planner agent
func NewPlannerAgent() *PlannerAgent {
	agent := &PlannerAgent{}

	// Try to load configuration from agent_configs.json
	configPath := "plans/agent_configs.json"
	configData, err := os.ReadFile(configPath)
	if err == nil {
		// Parse JSON data
		var configs []AgentConfig
		if err := json.Unmarshal(configData, &configs); err == nil {
			// Find the planner agent configuration
			for _, config := range configs {
				if config.Name == "planner" {
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
	agent.name = "planner"
	agent.role = "planner"
	agent.description = "Creates execution plans for programming tasks"
	agent.prompt = `You are a software architect and project planner. Your job is to break down programming tasks into executable steps.

Given a task description, create a detailed execution plan with the following information:

1. File structure needed
2. Dependencies to install
3. Implementation steps in order
4. Testing approach

CRITICAL RULES:
- For "execute" type steps, the description MUST be an ACTUAL executable shell command (e.g., "npm install express", "python main.py --param value", "curl -X GET https://api.example.com")
- NEVER write natural language instructions like "Use the search MCP server to..." - instead use actual commands
- If you need to search, use: curl or wget commands
- If you need to install packages, use: npm install, pip install, go get, etc.
- If you need to run code, use: python, node, go run, etc.
- IMPORTANT: This runs in a Linux Docker container. Use Linux commands like:
  - python3 instead of python
  - pip3 instead of pip
  - pip3 install --break-system-packages (for Alpine Python)
  - source venv/bin/activate (NOT venv\Scripts\activate)
  - use forward slashes / for paths
- PATH RULE: ALWAYS use RELATIVE paths, NEVER use absolute paths like /app/...
  - CORRECT: python3 script.py or python3 ./script.py
  - WRONG: python3 /app/script.py
- PARAMETER RULE: ALWAYS include ALL required parameters when running scripts or commands
  - For example, if a script requires --owner and --repo parameters, include them: python3 script.py --owner example --repo project
  - If a script can accept repository in format owner/repo, use that: python3 script.py --repo example/project
- When fetching data from APIs, first test if the endpoint works with simple curl -s. If the API requires headers or authentication, either skip that task or use jq to parse: curl -s URL | jq '.field'
- AVOID complex inline Python with pipes - write a small Python file instead and run it

FILENAME CONSISTENCY RULES:
- When planning code creation and execution steps, use EXACTLY the same filename in both steps
- For example, if you have a code step to create "script.py", the execute step must use "python3 script.py"
- NEVER change filenames between steps - this will cause execution failures

DEPENDENCY INSTALLATION RULES:
- ALWAYS include dependency installation steps BEFORE running code that requires them
- For Python projects: Add "pip3 install --break-system-packages <package>" steps BEFORE any "python3" execution steps
- For Node.js projects: Add "npm install" or "npm install <package>" steps BEFORE running the application
- Analyze the code requirements and install ALL needed dependencies in advance
- If a script imports/requires external packages, those MUST be installed first in a separate execute step

For each step, specify:
- Step number
- Type: code, execute, test, debug, or review
- Description: ACTUAL executable command or specific coding instruction
- Which agent should handle it (coder, executor, debugger, reviewer)

Format your response as a JSON array:
[
  {
    "order": 1,
    "type": "code",
    "description": "Create main application file with entry point",
    "agent": "coder"
  },
  {
    "order": 2,
    "type": "execute",
    "description": "npm install express",
    "agent": "executor"
  }
]

Available agents:
- coder: Writes and modifies code files
- executor: Runs shell commands (MUST be actual executable commands, not natural language)
- debugger: Fixes errors and bugs
- reviewer: Reviews code quality

Be thorough and consider edge cases. If the task is complex, break it into smaller subtasks.`

	return agent
}

// Execute runs the planner agent
func (a *PlannerAgent) Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error) {
	// Enhance input with context
	enhancedInput := a.enhanceInput(ctx, input)

	// Call LLM to generate plan
	response, err := a.callLLM(enhancedInput)
	if err != nil {
		return "", err
	}

	// Try to extract JSON plan
	plan := a.extractPlan(response)
	if plan == "" {
		// If no JSON found, return the raw response
		return response, nil
	}

	return plan, nil
}

// enhanceInput adds context to the input
func (a *PlannerAgent) enhanceInput(ctx *TaskContext, input string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Task: %s\n\n", input))

	if ctx.OS != "" {
		sb.WriteString(fmt.Sprintf("Operating System: %s\n", ctx.OS))
	}

	if len(ctx.Files) > 0 {
		sb.WriteString("\nExisting files:\n")
		for path := range ctx.Files {
			sb.WriteString(fmt.Sprintf("- %s\n", path))
		}
	}

	if len(ctx.AgentHistory) > 0 {
		sb.WriteString("\nPrevious execution history:\n")
		for _, exec := range ctx.AgentHistory {
			if exec.Error != "" {
				sb.WriteString(fmt.Sprintf("- %s failed: %s\n", exec.AgentName, exec.Error))
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(a.prompt)

	return sb.String()
}

// callLLM calls the language model
func (a *PlannerAgent) callLLM(input string) (string, error) {
	// Use the base agent's CallLLM method
	return a.CallLLM(input)
}

// extractPlan attempts to extract a JSON plan from the response
func (a *PlannerAgent) extractPlan(response string) string {
	// Look for JSON array in the response
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")

	if startIdx >= 0 && endIdx > startIdx {
		jsonStr := response[startIdx : endIdx+1]
		// Clean up invalid JSON characters before parsing
		// Remove any non-UTF8 or control characters
		cleanJsonStr := strings.Map(func(r rune) rune {
			if r < 0x20 && r != 0x0A && r != 0x0D && r != 0x09 {
				// Remove control characters except newline, carriage return, and tab
				return -1
			}
			return r
		}, jsonStr)
		// Validate it's valid JSON
		var plan []map[string]interface{}
		if err := json.Unmarshal([]byte(cleanJsonStr), &plan); err == nil {
			return cleanJsonStr
		}
	}

	return ""
}
