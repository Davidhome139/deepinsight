package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"backend/internal/config"
	"backend/internal/pkg/llm"
)

// Orchestrator manages multiple agents and coordinates task execution
type Orchestrator struct {
	agents        map[string]Agent
	activeTasks   map[string]*TaskContext
	mcpManager    *MCPManager
	skillRegistry *SkillRegistry
	llmClient     llm.Client
	mu            sync.RWMutex
	eventMu       sync.Mutex // Protects concurrent event handler calls
	eventHandlers []EventHandler

	// Enhanced components (P1-P3)
	codebaseIndexer      *CodebaseIndexer
	humanInLoop          *HumanInLoopManager
	multiFileCoordinator *MultiFileCoordinator
	gitManager           *GitOperationsManager
	lspManager           *LSPManager
	streamingEnabled     bool
	approvalRequired     bool
}

// TaskContext holds the state for an active task
type TaskContext struct {
	ID             string
	Description    string
	CurrentAgent   string
	Files          map[string]string
	ExecutionLog   []LogEntry
	WorkingDir     string
	OS             string
	UserID         uint
	Context        context.Context
	Cancel         context.CancelFunc
	AgentHistory   []AgentExecution
	IterationCount int
	MaxIterations  int
}

// LogEntry represents a log message
type LogEntry struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// AgentExecution tracks agent execution history
type AgentExecution struct {
	AgentName string    `json:"agent_name"`
	Input     string    `json:"input"`
	Output    string    `json:"output"`
	Error     string    `json:"error,omitempty"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// PlanStep represents a single step in an execution plan
type PlanStep struct {
	Order       int    `json:"order"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Agent       string `json:"agent"`
}

// EventHandler is called when orchestrator events occur
type EventHandler func(taskID string, eventType string, data interface{})

// Agent interface that all agents must implement
type Agent interface {
	Name() string
	Role() string
	Description() string
	Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error)
	GetPrompt() string
	SetPrompt(prompt string)
	SetLLMClient(client llm.Client)
}

// NewOrchestrator creates a new agent orchestrator
func NewOrchestrator(llmClient llm.Client, mcpManager *MCPManager, skillRegistry *SkillRegistry) *Orchestrator {
	workDir := "." // Default working directory

	o := &Orchestrator{
		agents:           make(map[string]Agent),
		activeTasks:      make(map[string]*TaskContext),
		mcpManager:       mcpManager,
		skillRegistry:    skillRegistry,
		llmClient:        llmClient,
		eventHandlers:    []EventHandler{},
		streamingEnabled: true,
		approvalRequired: false, // Can be enabled per task

		// Initialize enhanced components
		codebaseIndexer: NewCodebaseIndexer(workDir),
		humanInLoop:     NewHumanInLoopManager(5 * time.Minute),
		gitManager:      NewGitOperationsManager(workDir),
		lspManager:      NewLSPManager(workDir),
	}

	// Initialize multi-file coordinator with indexer
	o.multiFileCoordinator = NewMultiFileCoordinator(o.codebaseIndexer)

	// Register default agents with LLM client
	planner := NewPlannerAgent()
	planner.SetLLMClient(llmClient)
	o.RegisterAgent(planner)

	coder := NewCoderAgent()
	coder.SetLLMClient(llmClient)
	o.RegisterAgent(coder)

	debugger := NewDebuggerAgent()
	debugger.SetLLMClient(llmClient)
	o.RegisterAgent(debugger)

	executor := NewExecutorAgent()
	executor.SetLLMClient(llmClient)
	o.RegisterAgent(executor)

	reviewer := NewReviewerAgent()
	reviewer.SetLLMClient(llmClient)
	o.RegisterAgent(reviewer)

	// Register classifier agent
	classifier := NewClassifierAgent()
	classifier.SetLLMClient(llmClient)
	o.RegisterAgent(classifier)

	// Register enhanced agents (P2)
	reflexion := NewReflexionAgent()
	reflexion.SetLLMClient(llmClient)
	o.RegisterAgent(reflexion)

	tdd := NewTestDrivenGenerator()
	tdd.SetLLMClient(llmClient)
	o.RegisterAgent(tdd)

	// Load agent configurations from file if exists
	o.loadAgentConfigurations()

	// Auto-discover tools on startup
	go o.DiscoverTools()

	// Index codebase in background
	go func() {
		if err := o.codebaseIndexer.Index(); err != nil {
			fmt.Printf("[Orchestrator] Failed to index codebase: %v\n", err)
		} else {
			fmt.Println("[Orchestrator] Codebase indexed successfully")
		}
	}()

	return o
}

// RegisterAgent adds an agent to the orchestrator
func (o *Orchestrator) RegisterAgent(agent Agent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.agents[agent.Name()] = agent
}

// GetAgent retrieves an agent by name
func (o *Orchestrator) GetAgent(name string) (Agent, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	agent, ok := o.agents[name]
	return agent, ok
}

// ListAgents returns all registered agents
func (o *Orchestrator) ListAgents() []Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agents := make([]Agent, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}
	return agents
}

// AddEventHandler registers an event handler
func (o *Orchestrator) AddEventHandler(handler EventHandler) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.eventHandlers = append(o.eventHandlers, handler)
}

// emitEvent sends an event to all handlers
func (o *Orchestrator) emitEvent(taskID string, eventType string, data interface{}) {
	o.mu.RLock()
	handlers := make([]EventHandler, len(o.eventHandlers))
	copy(handlers, o.eventHandlers)
	o.mu.RUnlock()

	// Serialize event handler calls to prevent concurrent WebSocket writes
	o.eventMu.Lock()
	defer o.eventMu.Unlock()

	for _, handler := range handlers {
		// Each handler filters by taskID, so we pass the full event
		handler(taskID, eventType, data)
	}
}

// RemoveEventHandler removes a specific event handler
func (o *Orchestrator) RemoveEventHandler(handlerToRemove EventHandler) {
	o.mu.Lock()
	defer o.mu.Unlock()

	newHandlers := []EventHandler{}
	for _, handler := range o.eventHandlers {
		// Keep handlers that are not the one we're removing
		// We compare by pointer
		if fmt.Sprintf("%p", handler) != fmt.Sprintf("%p", handlerToRemove) {
			newHandlers = append(newHandlers, handler)
		}
	}
	o.eventHandlers = newHandlers
}

// StartTask begins executing a new task
func (o *Orchestrator) StartTask(taskID string, description string, initialFiles map[string]string, os string, workingDir string) (*TaskContext, error) {
	ctx, cancel := context.WithCancel(context.Background())

	task := &TaskContext{
		ID:            taskID,
		Description:   description,
		Files:         make(map[string]string), // Initialize as empty map, will be populated by WriteFile
		ExecutionLog:  []LogEntry{},
		WorkingDir:    workingDir,
		OS:            os,
		Context:       ctx,
		Cancel:        cancel,
		AgentHistory:  []AgentExecution{},
		MaxIterations: 10,
	}

	o.mu.Lock()
	o.activeTasks[taskID] = task
	o.mu.Unlock()

	// Persist initial files if any
	if initialFiles != nil {
		for filePath, content := range initialFiles {
			if err := o.WriteFile(taskID, filePath, content); err != nil {
				fmt.Printf("[Orchestrator] Failed to persist initial file %s: %v\n", filePath, err)
				// Continue processing other files even if one fails
			}
		}
	}

	// Start task execution in a goroutine
	go o.executeTask(task)

	return task, nil
}

// executeTask runs the main task execution loop
func (o *Orchestrator) executeTask(task *TaskContext) {
	fmt.Printf("[Orchestrator] executeTask started for task %s\n", task.ID)
	defer func() {
		fmt.Printf("[Orchestrator] executeTask completed for task %s\n", task.ID)
		o.mu.Lock()
		delete(o.activeTasks, task.ID)
		o.mu.Unlock()
	}()

	o.log(task, "info", fmt.Sprintf("Starting task: %s", task.Description))
	o.emitEvent(task.ID, "task_started", map[string]string{
		"taskId":      task.ID,
		"description": task.Description,
	})

	// Step 1: Classify the task
	classification, err := o.runAgent(task, "classifier", task.Description)
	if err != nil {
		o.log(task, "warning", fmt.Sprintf("Task classification failed: %v, proceeding with default planning\n", err))
		// Proceed with default planning if classification fails
		plan, err := o.runAgent(task, "planner", task.Description)
		if err != nil {
			o.log(task, "error", fmt.Sprintf("Planning failed: %v", err))
			o.emitEvent(task.ID, "task_error", map[string]string{"message": err.Error()})
			return
		}

		o.log(task, "success", "Plan created successfully")

		// Parse plan into steps
		steps := o.parsePlan(plan)
		o.log(task, "info", fmt.Sprintf("Executing %d steps", len(steps)))

		// Execute each step with iteration support
		for i, step := range steps {
			// Existing step execution code
			select {
			case <-task.Context.Done():
				o.log(task, "info", "Task cancelled")
				o.emitEvent(task.ID, "task_error", map[string]string{"message": "Task cancelled"})
				return
			default:
			}

			o.log(task, "info", fmt.Sprintf("Step %d/%d: %s", i+1, len(steps), step.Description))

			// Execute step with auto-retry on failure
			success := o.executeStepWithRetry(task, step)
			if !success {
				o.log(task, "error", fmt.Sprintf("Step %d failed after retries", i+1))
				// Try to replan
				if task.IterationCount < task.MaxIterations {
					o.log(task, "info", "Replanning...")
					newPlan, err := o.runAgent(task, "planner",
						fmt.Sprintf("Previous step failed: %s. Current task: %s. Please replan.", step.Description, task.Description))
					if err == nil {
						newSteps := o.parsePlan(newPlan)
						steps = append(newSteps, steps[i+1:]...)
						task.IterationCount++
						continue
					}
				}
			}
		}
	} else {
		// Classification succeeded, parse the result
		o.log(task, "success", fmt.Sprintf("Task classified: %s\n", classification))
		o.emitEvent(task.ID, "task_classified", map[string]string{
			"classification": classification,
		})

		// Check if the task is a simple task that can be completed in chat
		var taskClass TaskClassification
		if err := json.Unmarshal([]byte(classification), &taskClass); err == nil {
			if taskClass.IsSimpleTask {
				o.log(task, "info", "Task is a simple task, redirecting to chat execution")
				o.emitEvent(task.ID, "simple_task_detected", map[string]interface{}{
					"task_id":        task.ID,
					"description":    task.Description,
					"classification": taskClass,
					"message":        "This is a simple task that can be completed directly in chat.",
				})
				// Send task_complete event to trigger frontend redirect
				o.emitEvent(task.ID, "task_complete", map[string]interface{}{
					"task_id": task.ID,
					"status":  "completed",
					"message": "Task completed successfully.",
					"results": map[string]interface{}{},
				})
				return
			}
		}

		// Step 2: Planning based on classification
		plan, err := o.runAgent(task, "planner", task.Description+"\n\nTask classification: "+classification)
		if err != nil {
			o.log(task, "error", fmt.Sprintf("Planning failed: %v", err))
			o.emitEvent(task.ID, "task_error", map[string]string{"message": err.Error()})
			return
		}

		o.log(task, "success", "Plan created successfully")

		// Parse plan into steps
		steps := o.parsePlan(plan)
		o.log(task, "info", fmt.Sprintf("Executing %d steps", len(steps)))

		// Step 3: Execute each step with iteration support
		for i, step := range steps {
			select {
			case <-task.Context.Done():
				o.log(task, "info", "Task cancelled")
				o.emitEvent(task.ID, "task_error", map[string]string{"message": "Task cancelled"})
				return
			default:
			}

			o.log(task, "info", fmt.Sprintf("Step %d/%d: %s", i+1, len(steps), step.Description))

			// Execute step with auto-retry on failure
			success := o.executeStepWithRetry(task, step)
			if !success {
				o.log(task, "error", fmt.Sprintf("Step %d failed after retries", i+1))
				// Try to replan
				if task.IterationCount < task.MaxIterations {
					o.log(task, "info", "Replanning...")
					newPlan, err := o.runAgent(task, "planner",
						fmt.Sprintf("Previous step failed: %s. Current task: %s. Please replan.", step.Description, task.Description))
					if err == nil {
						newSteps := o.parsePlan(newPlan)
						steps = append(newSteps, steps[i+1:]...)
						task.IterationCount++
						continue
					}
				}
			}
		}
	}

	o.log(task, "success", "Task completed successfully!")

	// Save all agent configurations
	o.saveAgentConfigurations()

	// Create results directory and save task results
	if err := o.saveTaskResults(task); err != nil {
		o.log(task, "error", fmt.Sprintf("Failed to save task results: %v", err))
	}

	// Prepare task results to send to frontend
	results := map[string]interface{}{
		"task_id":         task.ID,
		"description":     task.Description,
		"start_time":      task.AgentHistory[0].StartTime,
		"end_time":        task.AgentHistory[len(task.AgentHistory)-1].EndTime,
		"duration":        task.AgentHistory[len(task.AgentHistory)-1].EndTime.Sub(task.AgentHistory[0].StartTime),
		"files_generated": len(task.Files),
		"iterations":      task.IterationCount,
		"files":           task.Files,
		"agent_history":   task.AgentHistory,
		"execution_log":   task.ExecutionLog,
	}

	o.emitEvent(task.ID, "task_complete", results)
}

// executeStepWithRetry executes a single step with retry logic
func (o *Orchestrator) executeStepWithRetry(task *TaskContext, step PlanStep) bool {
	maxRetries := 2

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			o.log(task, "info", fmt.Sprintf("Retry attempt %d/%d", attempt, maxRetries-1))
		}

		// For execute steps, check if the command references files that don't exist
		// and try to find matching files in task.Files
		if step.Type == "execute" && strings.Contains(step.Description, ".py") {
			o.checkAndFixFilenameMismatch(task, &step)
		}

		agent := o.selectAgentForStep(step)
		result, err := o.runAgent(task, agent.Name(), step.Description)

		if err == nil {
			o.log(task, "success", fmt.Sprintf("Step completed: %s", result))
			return true
		}

		o.log(task, "error", fmt.Sprintf("Step failed: %v", err))

		// Check if max task iterations reached
		if task.IterationCount >= task.MaxIterations {
			o.log(task, "error", "Max iterations reached, stopping task")
			o.emitEvent(task.ID, "task_error", map[string]string{"message": "Max iterations reached"})
			return false
		}
		task.IterationCount++

		// Try auto-fix on first failure only
		if attempt < maxRetries-1 {
			fixContext := fmt.Sprintf("Error: %v\nStep: %s\nAttempt: %d/%d", err, step.Description, attempt+1, maxRetries)
			fix, fixErr := o.runAgent(task, "debugger", fixContext)
			if fixErr == nil && fix != "" {
				o.log(task, "info", "Auto-fix suggestions received, applying fix")
				// Log fix content for debugging
				fmt.Printf("[Orchestrator] Debugger fix content: %s\n", fix)
				// Apply fix by checking if it contains code blocks for missing files
				o.applyDebuggerFix(task, fix)
			}
		}
	}

	return false
}

// runAgent executes a specific agent
func (o *Orchestrator) runAgent(task *TaskContext, agentName string, input string) (string, error) {
	fmt.Printf("[Orchestrator] runAgent: %s for task %s\n", agentName, task.ID)
	agent, ok := o.GetAgent(agentName)
	if !ok {
		return "", fmt.Errorf("agent %s not found", agentName)
	}

	task.CurrentAgent = agentName
	o.emitEvent(task.ID, "agent_started", map[string]interface{}{
		"id":     fmt.Sprintf("%s-%d", agentName, len(task.AgentHistory)),
		"name":   agent.Name(),
		"role":   agent.Role(),
		"action": input[:min(len(input), 100)],
	})

	execRecord := AgentExecution{
		AgentName: agentName,
		Input:     input,
		StartTime: time.Now(),
	}

	// Auto-discover and use relevant MCP tools and skills
	o.enhanceInputWithTools(&input, agentName)

	// Log the agent request
	inputPreview := input
	if len(inputPreview) > 500 {
		inputPreview = inputPreview[:500] + "..."
	}
	o.log(task, "agent", fmt.Sprintf("[%s] REQUEST:\n%s", agentName, inputPreview))

	fmt.Printf("[Orchestrator] Executing agent %s with input length: %d\n", agentName, len(input))
	result, err := agent.Execute(task, input, o.mcpManager, o.skillRegistry)
	fmt.Printf("[Orchestrator] Agent %s execution completed, result length: %d, error: %v\n", agentName, len(result), err)

	// Log the agent response
	resultPreview := result
	if len(resultPreview) > 500 {
		resultPreview = resultPreview[:500] + "..."
	}
	o.log(task, "agent", fmt.Sprintf("[%s] RESPONSE:\n%s", agentName, resultPreview))

	// After coder completes, write generated files to disk
	if agentName == "coder" && err == nil {
		fmt.Printf("[Orchestrator] Coder agent completed, checking for generated files...\n")
		fmt.Printf("[Orchestrator] Task.Files contains %d files\n", len(task.Files))
		if len(task.Files) > 0 {
			fmt.Printf("[Orchestrator] Writing %d files to disk for task %s\n", len(task.Files), task.ID)
			for filePath, content := range task.Files {
				fmt.Printf("[Orchestrator] Processing file: %s (content length: %d bytes)\n", filePath, len(content))
				if err := o.WriteFile(task.ID, filePath, content); err != nil {
					fmt.Printf("[Orchestrator] Failed to write file %s: %v\n", filePath, err)
				} else {
					fmt.Printf("[Orchestrator] Successfully wrote file: %s\n", filePath)
				}
			}
		} else {
			fmt.Printf("[Orchestrator] No files generated by coder agent\n")
		}
	}

	execRecord.EndTime = time.Now()
	execRecord.Output = result
	if err != nil {
		execRecord.Error = err.Error()
	}
	task.AgentHistory = append(task.AgentHistory, execRecord)

	status := "completed"
	errorMsg := ""
	if err != nil {
		status = "error"
		errorMsg = err.Error()
	}

	o.emitEvent(task.ID, "agent_completed", map[string]interface{}{
		"agentId":   fmt.Sprintf("%s-%d", agentName, len(task.AgentHistory)-1),
		"agentName": agent.Name(),
		"status":    status,
		"result":    result,
		"error":     errorMsg,
	})

	return result, err
}

// enhanceInputWithTools auto-discovers and adds relevant tools to input
func (o *Orchestrator) enhanceInputWithTools(input *string, agentName string) {
	// Get available MCPs
	mcps := o.mcpManager.ListConnected()
	if len(mcps) > 0 {
		mcpList := make([]string, len(mcps))
		for i, mcp := range mcps {
			mcpList[i] = mcp.Name
		}
		*input += fmt.Sprintf("\n\nAvailable MCP servers: %s", strings.Join(mcpList, ", "))
	}

	// Get relevant skills
	skills := o.skillRegistry.FindRelevant(*input)
	if len(skills) > 0 {
		skillList := make([]string, len(skills))
		for i, skill := range skills {
			skillList[i] = fmt.Sprintf("%s: %s", skill.Name(), skill.Description())
		}
		*input += fmt.Sprintf("\n\nAvailable skills: %s", strings.Join(skillList, "; "))
	}
}

// parsePlan extracts steps from the planner's output
func (o *Orchestrator) parsePlan(plan string) []PlanStep {
	// Try to parse as JSON first
	// Clean up invalid JSON characters before parsing
	// Remove any non-UTF8 or control characters
	cleanPlan := strings.Map(func(r rune) rune {
		if r < 0x20 && r != 0x0A && r != 0x0D && r != 0x09 {
			// Remove control characters except newline, carriage return, and tab
			return -1
		}
		return r
	}, plan)

	var steps []PlanStep
	if err := json.Unmarshal([]byte(cleanPlan), &steps); err == nil {
		return steps
	}

	// Fallback: parse from text format
	// Expected format: "1. [TYPE] Description"
	lines := strings.Split(plan, "\n")
	steps = []PlanStep{}
	order := 1

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to extract step info
		step := PlanStep{
			Order:       order,
			Type:        "code",
			Description: line,
			Agent:       "coder",
		}

		// Check for type markers
		if strings.Contains(line, "[EXEC]") || strings.Contains(line, "[RUN]") {
			step.Type = "execute"
			step.Agent = "executor"
		} else if strings.Contains(line, "[DEBUG]") || strings.Contains(line, "[FIX]") {
			step.Type = "debug"
			step.Agent = "debugger"
		} else if strings.Contains(line, "[TEST]") || strings.Contains(line, "[VERIFY]") {
			step.Type = "test"
			step.Agent = "executor"
		} else if strings.Contains(line, "[REVIEW]") {
			step.Type = "review"
			step.Agent = "reviewer"
		}

		steps = append(steps, step)
		order++
	}

	return steps
}

// selectAgentForStep chooses the appropriate agent for a step
func (o *Orchestrator) selectAgentForStep(step PlanStep) Agent {
	agentName := step.Agent
	if agentName == "" {
		switch step.Type {
		case "execute", "test":
			agentName = "executor"
		case "debug":
			agentName = "debugger"
		case "review":
			agentName = "reviewer"
		default:
			agentName = "coder"
		}
	}

	agent, _ := o.GetAgent(agentName)
	return agent
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(a, b string) int {
	m := len(a)
	n := len(b)
	matrix := make([][]int, m+1)

	for i := range matrix {
		matrix[i] = make([]int, n+1)
		matrix[i][0] = i
	}

	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			matrix[i][j] = min(matrix[i-1][j]+1, min(matrix[i][j-1]+1, matrix[i-1][j-1]+cost))
		}
	}

	return matrix[m][n]
}

// checkAndFixFilenameMismatch checks if the execute command references files that don't exist
// and tries to find matching files in task.Files
func (o *Orchestrator) checkAndFixFilenameMismatch(task *TaskContext, step *PlanStep) {
	// Extract filenames from the command

	// Regular expressions to find Python filenames in command
	// Note: Go regex doesn't support negative lookbehind (?<!...)
	regexPatterns := []string{
		`\bpython3?\s+([\w./-]+\.py)`,          // python3 filename.py
		`\bpython3?\s+(['"])([\w./-]+\.py)\\1`, // python3 "filename.py"
		`\bpython3?\s+\./([\w./-]+\.py)`,       // python3 ./filename.py
	}

	var referencedFile string
	var matchIndex int
	var regex *regexp.Regexp

	// Try each regex pattern to find a match
	for _, pattern := range regexPatterns {
		regex = regexp.MustCompile(pattern)
		matches := regex.FindStringSubmatch(step.Description)
		if len(matches) >= 2 {
			// Determine which group contains the filename based on the pattern
			if len(matches) >= 3 && matches[2] != "" {
				matchIndex = 2 // For patterns with quotes
			} else {
				matchIndex = 1 // For patterns without quotes
			}
			referencedFile = matches[matchIndex]
			break
		}
	}

	if referencedFile == "" {
		return // No Python filenames found
	}

	// Remove leading ./ if present
	referencedFile = strings.TrimPrefix(referencedFile, "./")

	// Check if the file exists in task.Files
	if _, exists := task.Files[referencedFile]; exists {
		return // File exists, no fix needed
	}

	// Find the most similar Python file in task.Files
	var bestMatch string
	var bestDistance int = -1

	for file := range task.Files {
		if strings.HasSuffix(file, ".py") {
			// Calculate similarity using Levenshtein distance
			distance := levenshteinDistance(referencedFile, file)

			// Update best match if this is the first file or better match
			if bestMatch == "" || distance < bestDistance {
				bestMatch = file
				bestDistance = distance
			}
		}
	}

	// If a similar file is found, update the command
	if bestMatch != "" {
		o.log(task, "info", fmt.Sprintf("Found filename mismatch: command references '%s' but actual file is '%s'. Fixing...", referencedFile, bestMatch))

		// Replace the filename in the command - use a more flexible approach
		// First, find any python command pattern
		pythonRe := regexp.MustCompile(`(\bpython3?\s+['"]?)([\w./-]+\.py)(['"]?)`)
		// Replace with the correct filename
		step.Description = pythonRe.ReplaceAllString(step.Description, "${1}"+bestMatch+"${3}")

		o.log(task, "info", fmt.Sprintf("Updated command: %s", step.Description))
	} else {
		o.log(task, "warning", fmt.Sprintf("Command references '%s' but no Python files found in generated files", referencedFile))
		// List all files to help debug
		fileList := make([]string, 0, len(task.Files))
		for file := range task.Files {
			fileList = append(fileList, file)
		}
		o.log(task, "info", fmt.Sprintf("Available files: %v", fileList))
	}
}

// ExecuteCommand runs a shell command in the task's working directory
func (o *Orchestrator) ExecuteCommand(taskID string, command string) (string, error) {
	o.mu.RLock()
	task, exists := o.activeTasks[taskID]
	o.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("task not found")
	}

	fmt.Printf("[Orchestrator] ExecuteCommand: Task ID: %s\n", taskID)
	fmt.Printf("[Orchestrator] ExecuteCommand: Command: %s\n", command)
	fmt.Printf("[Orchestrator] ExecuteCommand: Task working directory: %s\n", task.WorkingDir)
	fmt.Printf("[Orchestrator] ExecuteCommand: OS: %s\n", task.OS)

	// List working directory contents before execution
	fmt.Printf("[Orchestrator] ExecuteCommand: Listing files in %s:\n", task.WorkingDir)
	files, err := os.ReadDir(task.WorkingDir)
	if err != nil {
		fmt.Printf("[Orchestrator] ExecuteCommand: Failed to list directory: %v\n", err)
	} else {
		for _, file := range files {
			fmt.Printf("[Orchestrator] ExecuteCommand:   %s (%d bytes)\n", file.Name(), getFileSize(filepath.Join(task.WorkingDir, file.Name())))
		}
	}

	var cmd *exec.Cmd
	if task.OS == "windows" {
		fmt.Printf("[Orchestrator] ExecuteCommand: Using cmd.exe\n")
		cmd = exec.CommandContext(task.Context, "cmd", "/c", command)
	} else {
		fmt.Printf("[Orchestrator] ExecuteCommand: Using sh\n")
		cmd = exec.CommandContext(task.Context, "sh", "-c", command)
	}

	cmd.Dir = task.WorkingDir
	cmd.Env = os.Environ()
	fmt.Printf("[Orchestrator] ExecuteCommand: Command directory set to: %s\n", cmd.Dir)

	o.log(task, "command", command)

	fmt.Printf("[Orchestrator] ExecuteCommand: Starting command execution\n")
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	executionTime := time.Since(startTime)
	outputStr := string(output)

	fmt.Printf("[Orchestrator] ExecuteCommand: Command execution completed in %v\n", executionTime)
	fmt.Printf("[Orchestrator] ExecuteCommand: Exit code: %v\n", err)
	fmt.Printf("[Orchestrator] ExecuteCommand: Output (%d bytes): %s\n", len(outputStr), outputStr)

	if err != nil {
		o.log(task, "error", fmt.Sprintf("Command failed: %v\n%s", err, outputStr))
	} else {
		o.log(task, "success", "Command executed successfully")
	}

	o.emitEvent(task.ID, "terminal_output", map[string]string{
		"output": outputStr,
	})

	return outputStr, err
}

// getFileSize returns the size of a file in bytes
func getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return -1
	}
	return info.Size()
}

// ReadFile reads a file from the task's working directory
func (o *Orchestrator) ReadFile(taskID string, filePath string) (string, error) {
	o.mu.RLock()
	task, exists := o.activeTasks[taskID]
	o.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("task not found")
	}

	fullPath := filepath.Join(task.WorkingDir, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// WriteFile writes content to a file in the task's working directory
// getPersistenceDir returns the appropriate persistence directory based on the OS type
func (o *Orchestrator) getPersistenceDir() string {
	osType := config.GlobalConfig.OS
	if osType == "windows" {
		return "D:\\apps\\plans"
	}
	// For Linux and other systems
	return filepath.Join(os.Getenv("HOME"), "apps", "plans")
}

func (o *Orchestrator) WriteFile(taskID string, filePath string, content string) error {
	o.mu.RLock()
	task, exists := o.activeTasks[taskID]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task not found")
	}

	// Write to working directory (existing functionality)
	workingPath := filepath.Join(task.WorkingDir, filePath)
	dir := filepath.Dir(workingPath)
	fmt.Printf("[Orchestrator] WriteFile: Writing to working path: %s\n", workingPath)
	fmt.Printf("[Orchestrator] WriteFile: Task working directory: %s\n", task.WorkingDir)
	fmt.Printf("[Orchestrator] WriteFile: File path: %s\n", filePath)
	fmt.Printf("[Orchestrator] WriteFile: Content length: %d bytes\n", len(content))

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("[Orchestrator] WriteFile: Directory %s does not exist, creating...\n", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("[Orchestrator] WriteFile: Failed to create directory %s: %v\n", dir, err)
			return err
		}
		fmt.Printf("[Orchestrator] WriteFile: Successfully created directory %s\n", dir)
	} else if err != nil {
		fmt.Printf("[Orchestrator] WriteFile: Error checking directory %s: %v\n", dir, err)
	}

	// Write the file
	if err := os.WriteFile(workingPath, []byte(content), 0644); err != nil {
		fmt.Printf("[Orchestrator] WriteFile: Failed to write to %s: %v\n", workingPath, err)
		return err
	}
	fmt.Printf("[Orchestrator] WriteFile: Successfully wrote to %s\n", workingPath)

	// Verify the file was written
	if _, err := os.Stat(workingPath); os.IsNotExist(err) {
		fmt.Printf("[Orchestrator] WriteFile: ERROR: File %s was not created after write operation!\n", workingPath)
	} else {
		fmt.Printf("[Orchestrator] WriteFile: VERIFIED: File %s exists after write operation\n", workingPath)
	}

	// Check file content
	writtenContent, err := os.ReadFile(workingPath)
	if err != nil {
		fmt.Printf("[Orchestrator] WriteFile: ERROR: Failed to read back file %s: %v\n", workingPath, err)
	} else {
		fmt.Printf("[Orchestrator] WriteFile: VERIFIED: File content length: %d bytes\n", len(writtenContent))
		if string(writtenContent) == content {
			fmt.Printf("[Orchestrator] WriteFile: VERIFIED: File content matches expected\n")
		} else {
			fmt.Printf("[Orchestrator] WriteFile: WARNING: File content differs from expected\n")
		}
	}

	// Update task files
	task.Files[filePath] = content

	// Persist to the plans directory in the current working directory
	persistenceDir := "plans"
	// Create a task-specific subdirectory for organization
	taskPersistenceDir := filepath.Join(persistenceDir, taskID)
	persistPath := filepath.Join(taskPersistenceDir, filePath)

	// Ensure persistence directory exists
	persistFileDir := filepath.Dir(persistPath)
	if err := os.MkdirAll(persistFileDir, 0755); err != nil {
		fmt.Printf("[Orchestrator] Failed to create persistence directory %s: %v\n", persistFileDir, err)
		// Continue execution even if persistence fails, don't break the main workflow
	} else {
		// Write to persistence directory
		if err := os.WriteFile(persistPath, []byte(content), 0644); err != nil {
			fmt.Printf("[Orchestrator] Failed to persist file %s: %v\n", persistPath, err)
			// Continue execution even if persistence fails
		} else {
			fmt.Printf("[Orchestrator] Persisted file: %s\n", persistPath)
		}
	}

	// Detect language
	ext := filepath.Ext(filePath)
	language := "plaintext"
	langMap := map[string]string{
		".js": "javascript", ".ts": "typescript", ".py": "python",
		".go": "go", ".java": "java", ".json": "json", ".md": "markdown",
		".vue": "html", ".html": "html", ".css": "css", ".scss": "scss",
	}
	if l, ok := langMap[ext]; ok {
		language = l
	}

	o.emitEvent(task.ID, "code_update", map[string]string{
		"filePath": filePath,
		"content":  content,
		"language": language,
	})

	return nil
}

// GetFileTree returns the file tree for a task
func (o *Orchestrator) GetFileTree(taskID string) ([]FileNode, error) {
	o.mu.RLock()
	task, exists := o.activeTasks[taskID]
	o.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	return o.buildFileTree(task.WorkingDir)
}

// FileNode represents a file or directory in the tree
type FileNode struct {
	Label       string     `json:"label"`
	Path        string     `json:"path"`
	IsDirectory bool       `json:"isDirectory"`
	Children    []FileNode `json:"children,omitempty"`
}

// buildFileTree builds a file tree from a directory
func (o *Orchestrator) buildFileTree(root string) ([]FileNode, error) {
	var nodes []FileNode

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		// Skip hidden files and common ignore patterns
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "__pycache__" {
			continue
		}

		node := FileNode{
			Label:       name,
			Path:        filepath.Join(root, name),
			IsDirectory: entry.IsDir(),
		}

		if entry.IsDir() {
			children, err := o.buildFileTree(node.Path)
			if err == nil {
				node.Children = children
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetAgentConfigs returns all agent configurations
func (o *Orchestrator) GetAgentConfigs() []AgentConfig {
	agents := o.ListAgents()
	configs := make([]AgentConfig, len(agents))
	for i, agent := range agents {
		configs[i] = AgentConfig{
			Name:        agent.Name(),
			Role:        agent.Role(),
			Description: agent.Description(),
			Prompt:      agent.GetPrompt(),
		}
	}
	return configs
}

// UpdateAgentConfig updates an agent's configuration
func (o *Orchestrator) UpdateAgentConfig(name string, prompt string) error {
	agent, ok := o.GetAgent(name)
	if !ok {
		return fmt.Errorf("agent %s not found", name)
	}
	agent.SetPrompt(prompt)

	// Save configurations immediately when updated
	o.saveAgentConfigurations()
	return nil
}

// DiscoverTools triggers discovery of MCPs and skills
// Only runs once - use RefreshTools() to force re-discovery
func (o *Orchestrator) DiscoverTools() {
	// Discover MCPs (cached after first run)
	o.mcpManager.Discover()

	// Discover skills (cached after first run)
	o.skillRegistry.Discover()
}

// RefreshTools forces re-discovery of MCPs and skills
func (o *Orchestrator) RefreshTools() {
	fmt.Println("[Orchestrator] Refreshing all tools...")

	// Force refresh MCPs
	o.mcpManager.Refresh()

	// Force refresh skills
	o.skillRegistry.Refresh()
}

// SetModel changes the default LLM model
func (o *Orchestrator) SetModel(model string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if client, ok := o.llmClient.(*llm.AIClient); ok {
		client.SetModel(model)
		fmt.Printf("[Orchestrator] Model changed to: %s\n", model)
	}
}

// SetCurrentUser sets the current user ID for API key lookup
func (o *Orchestrator) SetCurrentUser(userID uint) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if client, ok := o.llmClient.(*llm.AIClient); ok {
		client.SetUserID(userID)
		fmt.Printf("[Orchestrator] Current user set to: %d\n", userID)
	}
}

// GetModel returns the current LLM model
func (o *Orchestrator) GetModel() string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if client, ok := o.llmClient.(*llm.AIClient); ok {
		return client.GetDefaultModel()
	}
	return ""
}

// GetActiveTask returns an active task by ID
func (o *Orchestrator) GetActiveTask(taskID string) (*TaskContext, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	task, ok := o.activeTasks[taskID]
	return task, ok
}

// log adds a log entry to the task
func (o *Orchestrator) log(task *TaskContext, logType string, message string) {
	entry := LogEntry{
		Type:      logType,
		Message:   message,
		Timestamp: time.Now(),
	}
	task.ExecutionLog = append(task.ExecutionLog, entry)

	o.emitEvent(task.ID, "log", map[string]interface{}{
		"type":      logType,
		"message":   message,
		"timestamp": entry.Timestamp.Unix(),
	})
}

// StopTask cancels an active task
func (o *Orchestrator) StopTask(taskID string) error {
	o.mu.RLock()
	task, exists := o.activeTasks[taskID]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task not found")
	}

	// Persist all files before stopping task
	if len(task.Files) > 0 {
		fmt.Printf("[Orchestrator] Persisting %d files before stopping task %s\n", len(task.Files), taskID)
		for filePath, content := range task.Files {
			if err := o.WriteFile(taskID, filePath, content); err != nil {
				fmt.Printf("[Orchestrator] Failed to persist file %s: %v\n", filePath, err)
			} else {
				fmt.Printf("[Orchestrator] Persisted file: %s\n", filePath)
			}
		}
	}

	// Save task results before stopping
	if err := o.saveTaskResults(task); err != nil {
		fmt.Printf("[Orchestrator] Failed to save task results: %v\n", err)
	}

	task.Cancel()
	return nil
}

// GetMCPManager returns the MCP manager
func (o *Orchestrator) GetMCPManager() *MCPManager {
	return o.mcpManager
}

// GetSkillRegistry returns the skill registry
func (o *Orchestrator) GetSkillRegistry() *SkillRegistry {
	return o.skillRegistry
}

// applyDebuggerFix applies fix suggestions from the debugger agent
func (o *Orchestrator) applyDebuggerFix(task *TaskContext, fix string) error {
	// Extract code blocks from the fix response
	codeBlocks := o.extractCodeBlocksFromFix(fix)
	if len(codeBlocks) == 0 {
		return nil // No code blocks to apply
	}

	// Write files to disk
	for _, block := range codeBlocks {
		if err := o.WriteFile(task.ID, block.FilePath, block.Code); err != nil {
			o.log(task, "error", fmt.Sprintf("Failed to apply fix for file %s: %v", block.FilePath, err))
		} else {
			o.log(task, "success", fmt.Sprintf("Applied fix for file: %s", block.FilePath))
		}
	}

	return nil
}

// extractCodeBlocksFromFix extracts code blocks from debugger fix response
func (o *Orchestrator) extractCodeBlocksFromFix(fix string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(fix, "\n")

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
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				// Opening - extract language
				lang := strings.Trim(trimmed, "`")
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

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AgentConfig represents agent configuration for serialization
type AgentConfig struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}

// loadAgentConfigurations loads agent configurations from a file
func (o *Orchestrator) loadAgentConfigurations() {
	// Create plans directory if it doesn't exist
	plansDir := "plans"
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		fmt.Printf("[Orchestrator] Failed to create plans directory: %v\n", err)
		return
	}

	// Check if agent configurations file exists
	configPath := filepath.Join(plansDir, "agent_configs.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("[Orchestrator] Agent configurations file not found at %s, using defaults\n", configPath)

		// Save default configurations to file
		fmt.Printf("[Orchestrator] Saving default agent configurations to %s\n", configPath)
		o.saveAgentConfigurations()
		return
	}

	// Read agent configurations from JSON file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("[Orchestrator] Failed to read agent configurations: %v\n", err)
		return
	}

	// Parse JSON data
	var configs []AgentConfig
	if err := json.Unmarshal(configData, &configs); err != nil {
		fmt.Printf("[Orchestrator] Failed to parse agent configurations: %v\n", err)
		return
	}

	// Update agent prompts
	for _, config := range configs {
		if err := o.UpdateAgentConfig(config.Name, config.Prompt); err != nil {
			fmt.Printf("[Orchestrator] Failed to update agent %s: %v\n", config.Name, err)
		}
	}

	fmt.Printf("[Orchestrator] Agent configurations loaded from %s\n", configPath)
}

// saveAgentConfigurations saves all agent configurations to a file
func (o *Orchestrator) saveAgentConfigurations() {
	// Get all agent configurations
	configs := o.GetAgentConfigs()

	// Create plans directory
	plansDir := "plans"
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		fmt.Printf("[Orchestrator] Failed to create plans directory: %v\n", err)
		return
	}

	// Save agent configurations to JSON file in plans directory
	configPath := filepath.Join(plansDir, "agent_configs.json")
	configData, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		fmt.Printf("[Orchestrator] Failed to marshal agent configurations: %v\n", err)
		return
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		fmt.Printf("[Orchestrator] Failed to save agent configurations: %v\n", err)
		return
	}

	fmt.Printf("[Orchestrator] Agent configurations saved to %s\n", configPath)
}

// saveTaskResults saves task results and output
func (o *Orchestrator) saveTaskResults(task *TaskContext) error {
	// Create a plans directory in the current working directory
	plansDir := "plans"
	// Create a task-specific directory for results
	resultsDir := filepath.Join(plansDir, task.ID, "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	// Save task summary
	summary := map[string]interface{}{
		"task_id":         task.ID,
		"description":     task.Description,
		"start_time":      task.AgentHistory[0].StartTime,
		"end_time":        task.AgentHistory[len(task.AgentHistory)-1].EndTime,
		"duration":        task.AgentHistory[len(task.AgentHistory)-1].EndTime.Sub(task.AgentHistory[0].StartTime),
		"files_generated": len(task.Files),
		"iterations":      task.IterationCount,
	}

	summaryPath := filepath.Join(resultsDir, fmt.Sprintf("%s_summary.json", task.ID))
	summaryData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task summary: %w", err)
	}

	if err := os.WriteFile(summaryPath, summaryData, 0644); err != nil {
		return fmt.Errorf("failed to save task summary: %w", err)
	}

	// Save agent execution history
	historyPath := filepath.Join(resultsDir, fmt.Sprintf("%s_history.json", task.ID))
	historyData, err := json.MarshalIndent(task.AgentHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent history: %w", err)
	}

	if err := os.WriteFile(historyPath, historyData, 0644); err != nil {
		return fmt.Errorf("failed to save agent history: %w", err)
	}

	// Save execution log
	logPath := filepath.Join(resultsDir, fmt.Sprintf("%s_log.json", task.ID))
	logData, err := json.MarshalIndent(task.ExecutionLog, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution log: %w", err)
	}

	if err := os.WriteFile(logPath, logData, 0644); err != nil {
		return fmt.Errorf("failed to save execution log: %w", err)
	}

	// Create a directory for generated files
	generatedFilesDir := filepath.Join(resultsDir, fmt.Sprintf("%s_files", task.ID))
	if err := os.MkdirAll(generatedFilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create generated files directory: %w", err)
	}

	// Save actual file contents to the generated files directory
	for filePath, content := range task.Files {
		// Create the directory structure if needed
		fileDir := filepath.Dir(filePath)
		if fileDir != "" {
			if err := os.MkdirAll(filepath.Join(generatedFilesDir, fileDir), 0755); err != nil {
				return fmt.Errorf("failed to create directory structure for file %s: %w", filePath, err)
			}
		}

		// Save the file content
		outputFilePath := filepath.Join(generatedFilesDir, filePath)
		if err := os.WriteFile(outputFilePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to save file %s: %w", filePath, err)
		}
	}

	// Create a simple text output file with task results
	outputPath := filepath.Join(resultsDir, fmt.Sprintf("%s_output.txt", task.ID))
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Task ID: %s\n", task.ID))
	output.WriteString(fmt.Sprintf("Description: %s\n", task.Description))
	output.WriteString(fmt.Sprintf("Status: Completed\n"))
	output.WriteString(fmt.Sprintf("Duration: %v\n", task.AgentHistory[len(task.AgentHistory)-1].EndTime.Sub(task.AgentHistory[0].StartTime)))
	output.WriteString(fmt.Sprintf("Files Generated: %d\n", len(task.Files)))
	output.WriteString(fmt.Sprintf("Generated Files Directory: %s\n", generatedFilesDir))
	output.WriteString("\nFiles:\n")

	for filePath := range task.Files {
		output.WriteString(fmt.Sprintf("- %s\n", filePath))
	}

	output.WriteString("\nAgent Execution Summary:\n")
	for i, exec := range task.AgentHistory {
		output.WriteString(fmt.Sprintf("%d. %s - %v\n", i+1, exec.AgentName, exec.EndTime.Sub(exec.StartTime)))
		if exec.Error != "" {
			output.WriteString(fmt.Sprintf("   Error: %s\n", exec.Error))
		}
	}

	if err := os.WriteFile(outputPath, []byte(output.String()), 0644); err != nil {
		return fmt.Errorf("failed to save task output: %w", err)
	}

	fmt.Printf("[Orchestrator] Task results saved to %s directory\n", resultsDir)
	return nil
}

// =====================================================
// Enhanced Methods (Priority 1-3 Features)
// =====================================================

// GetCodebaseContext returns relevant codebase context for a task
func (o *Orchestrator) GetCodebaseContext(task string, maxFiles int) *CodebaseContext {
	if o.codebaseIndexer == nil {
		return nil
	}
	return o.codebaseIndexer.GetContext(task, maxFiles)
}

// RequestFileWriteApproval requests approval before writing a file
func (o *Orchestrator) RequestFileWriteApproval(taskID, filePath, content string) (*ApprovalResponse, error) {
	if !o.approvalRequired || o.humanInLoop == nil {
		// Auto-approve if approval not required
		return &ApprovalResponse{
			Approved: true,
			Comment:  "auto-approved",
		}, nil
	}

	// Read existing content for diff
	existingContent := ""
	if data, err := os.ReadFile(filePath); err == nil {
		existingContent = string(data)
	}

	// Generate diff
	diff := generateSimpleDiff(existingContent, content)

	req := &ApprovalRequest{
		ID:          fmt.Sprintf("approval-%s-%d", taskID, time.Now().UnixNano()),
		TaskID:      taskID,
		Type:        "file_write",
		Description: fmt.Sprintf("Write file: %s", filePath),
		FilePath:    filePath,
		Content:     content,
		Diff:        diff,
		Details: map[string]interface{}{
			"lines_added":   strings.Count(content, "\n"),
			"lines_removed": strings.Count(existingContent, "\n"),
		},
	}

	// Emit approval request event
	o.emitEvent(taskID, "approval_required", req)

	// Wait for response
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return o.humanInLoop.RequestApproval(ctx, req)
}

// SubmitApproval submits an approval response
func (o *Orchestrator) SubmitApproval(response ApprovalResponse) error {
	if o.humanInLoop == nil {
		return fmt.Errorf("human-in-loop manager not initialized")
	}
	return o.humanInLoop.SubmitApproval(response)
}

// GetPendingApprovals returns all pending approval requests
func (o *Orchestrator) GetPendingApprovals() []*ApprovalRequest {
	if o.humanInLoop == nil {
		return nil
	}
	return o.humanInLoop.GetPendingApprovals()
}

// SetApprovalRequired enables or disables approval requirements
func (o *Orchestrator) SetApprovalRequired(required bool) {
	o.approvalRequired = required
}

// ProposeFileChange proposes a file change without immediate write
func (o *Orchestrator) ProposeFileChange(filePath, oldContent, newContent, reason string) error {
	if o.multiFileCoordinator == nil {
		return fmt.Errorf("multi-file coordinator not initialized")
	}

	changeType := "modify"
	if oldContent == "" {
		changeType = "create"
	}

	change := &FileChange{
		FilePath:   filePath,
		ChangeType: changeType,
		OldContent: oldContent,
		NewContent: newContent,
		Status:     "pending",
		Metadata: ChangeMetadata{
			Reason: reason,
		},
	}

	return o.multiFileCoordinator.ProposeChange(change)
}

// GetProposedChanges returns all proposed changes
func (o *Orchestrator) GetProposedChanges() []*FileChange {
	if o.multiFileCoordinator == nil {
		return nil
	}
	return o.multiFileCoordinator.GetChangePlan()
}

// GetChangesSummary returns a summary of proposed changes
func (o *Orchestrator) GetChangesSummary() *ChangesSummary {
	if o.multiFileCoordinator == nil {
		return nil
	}
	return o.multiFileCoordinator.GetChangesSummary()
}

// GetUnifiedDiff returns a unified diff of all proposed changes
func (o *Orchestrator) GetUnifiedDiff() string {
	if o.multiFileCoordinator == nil {
		return ""
	}
	return o.multiFileCoordinator.GetUnifiedDiff()
}

// ApproveAllChanges approves all pending changes
func (o *Orchestrator) ApproveAllChanges() {
	if o.multiFileCoordinator != nil {
		o.multiFileCoordinator.ApproveAll()
	}
}

// CreateStreamingGenerator creates a streaming code generator with event callback
func (o *Orchestrator) CreateStreamingGenerator(taskID string) *StreamingCodeGenerator {
	return NewStreamingCodeGenerator(func(event CodeStreamEvent) {
		o.emitEvent(taskID, "code_stream", event)
	})
}

// Git Operations

// CreateTaskBranch creates a Git branch for a task
func (o *Orchestrator) CreateTaskBranch(taskID, description string) error {
	if o.gitManager == nil {
		return fmt.Errorf("git manager not initialized")
	}
	return o.gitManager.CreateTaskBranch(taskID, description)
}

// CommitChanges commits all changes with a message
func (o *Orchestrator) CommitChanges(taskID, message string) (*GitCommit, error) {
	if o.gitManager == nil {
		return nil, fmt.Errorf("git manager not initialized")
	}

	if err := o.gitManager.StageAll(); err != nil {
		return nil, fmt.Errorf("failed to stage changes: %v", err)
	}

	return o.gitManager.Commit(message, taskID)
}

// GetGitStatus returns the current Git status
func (o *Orchestrator) GetGitStatus() (*GitStatus, error) {
	if o.gitManager == nil {
		return nil, fmt.Errorf("git manager not initialized")
	}
	return o.gitManager.GetStatus()
}

// PreparePullRequest prepares PR information for the task
func (o *Orchestrator) PreparePullRequest(taskID, description string) (*PRInfo, error) {
	if o.gitManager == nil {
		return nil, fmt.Errorf("git manager not initialized")
	}
	return o.gitManager.PreparePRInfo(taskID, description)
}

// LSP Operations

// CheckSyntax checks syntax for a file
func (o *Orchestrator) CheckSyntax(filePath string) ([]Diagnostic, error) {
	if o.lspManager == nil {
		return nil, fmt.Errorf("LSP manager not initialized")
	}
	return o.lspManager.CheckSyntax(filePath)
}

// CheckAllSyntax checks syntax for all files in workspace
func (o *Orchestrator) CheckAllSyntax() (map[string][]Diagnostic, error) {
	if o.lspManager == nil {
		return nil, fmt.Errorf("LSP manager not initialized")
	}
	return o.lspManager.CheckAllFiles()
}

// ReindexCodebase triggers a re-index of the codebase
func (o *Orchestrator) ReindexCodebase() error {
	if o.codebaseIndexer == nil {
		return fmt.Errorf("codebase indexer not initialized")
	}
	return o.codebaseIndexer.Index()
}

// GetAffectedFiles returns files affected by changing a symbol
func (o *Orchestrator) GetAffectedFiles(symbolName string) []string {
	if o.codebaseIndexer == nil {
		return nil
	}
	return o.codebaseIndexer.GetAffectedFiles(symbolName)
}

// GetFileContent returns file content with optional line range
func (o *Orchestrator) GetFileContent(filePath string, startLine, endLine int) (string, error) {
	if o.codebaseIndexer == nil {
		return "", fmt.Errorf("codebase indexer not initialized")
	}
	return o.codebaseIndexer.GetFileContent(filePath, startLine, endLine)
}

// Helper function to generate a simple diff
func generateSimpleDiff(oldContent, newContent string) string {
	if oldContent == "" {
		return fmt.Sprintf("+++ New file\n%s", newContent)
	}

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var diff strings.Builder
	diff.WriteString("--- a/file\n")
	diff.WriteString("+++ b/file\n")

	// Simple line comparison
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	for i := 0; i < maxLines; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			if oldLine != "" {
				diff.WriteString(fmt.Sprintf("-%s\n", oldLine))
			}
			if newLine != "" {
				diff.WriteString(fmt.Sprintf("+%s\n", newLine))
			}
		}
	}

	return diff.String()
}
