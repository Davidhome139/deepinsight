package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ReflexionAgent implements the Reflexion self-improvement pattern
// It learns from failures and maintains a memory of past experiences
type ReflexionAgent struct {
	baseAgent
	memory         *ReflexionMemory
	maxReflections int
	learningRate   float64
}

// ReflexionMemory stores experiences for self-improvement
type ReflexionMemory struct {
	Experiences     []Experience     `json:"experiences"`
	SuccessPatterns []SuccessPattern `json:"success_patterns"`
	FailurePatterns []FailurePattern `json:"failure_patterns"`
	Lessons         []Lesson         `json:"lessons"`
	mu              sync.RWMutex
}

// Experience represents a single execution experience
type Experience struct {
	TaskID      string    `json:"task_id"`
	Input       string    `json:"input"`
	Output      string    `json:"output"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Duration    float64   `json:"duration"`
	Timestamp   time.Time `json:"timestamp"`
	Reflection  string    `json:"reflection,omitempty"`
	Improvement string    `json:"improvement,omitempty"`
}

// SuccessPattern represents a pattern that led to success
type SuccessPattern struct {
	Pattern     string   `json:"pattern"`
	Contexts    []string `json:"contexts"`
	SuccessRate float64  `json:"success_rate"`
	Count       int      `json:"count"`
}

// FailurePattern represents a pattern that led to failure
type FailurePattern struct {
	Pattern    string   `json:"pattern"`
	Causes     []string `json:"causes"`
	Solutions  []string `json:"solutions"`
	FailCount  int      `json:"fail_count"`
	FixedCount int      `json:"fixed_count"`
}

// Lesson represents a learned lesson from experience
type Lesson struct {
	ID           string    `json:"id"`
	Description  string    `json:"description"`
	Context      string    `json:"context"`
	Action       string    `json:"action"`
	Outcome      string    `json:"outcome"`
	LearnedAt    time.Time `json:"learned_at"`
	AppliedCount int       `json:"applied_count"`
}

// ReflexionResult represents the result of a reflection cycle
type ReflexionResult struct {
	OriginalOutput string   `json:"original_output"`
	Reflection     string   `json:"reflection"`
	Improvements   []string `json:"improvements"`
	RevisedOutput  string   `json:"revised_output"`
	Iteration      int      `json:"iteration"`
	Converged      bool     `json:"converged"`
}

// NewReflexionAgent creates a new Reflexion agent
func NewReflexionAgent() *ReflexionAgent {
	return &ReflexionAgent{
		baseAgent: baseAgent{
			name:        "reflexion",
			role:        "self-improver",
			description: "Self-improving agent that learns from failures using the Reflexion pattern",
			prompt: `You are a Reflexion agent that improves through self-reflection.

When given a task result, you:
1. Analyze if the result is correct and complete
2. Identify any errors, inefficiencies, or missing elements
3. Reflect on what could be improved
4. Generate an improved version

REFLECTION PROCESS:
1. EVALUATE: Is the output correct? Complete? Efficient?
2. IDENTIFY: What specific issues exist?
3. ANALYZE: Why did these issues occur?
4. IMPROVE: How can we fix them?
5. APPLY: Generate the improved output

OUTPUT FORMAT:
{
  "evaluation": {
    "correct": true/false,
    "complete": true/false,
    "efficient": true/false,
    "issues": ["issue1", "issue2"]
  },
  "reflection": "Your analysis of what went wrong and why",
  "improvements": ["improvement1", "improvement2"],
  "revised_output": "The improved output",
  "lesson_learned": "A generalizable lesson from this experience"
}`,
		},
		memory:         NewReflexionMemory(),
		maxReflections: 3,
		learningRate:   0.1,
	}
}

// NewReflexionMemory creates a new memory store
func NewReflexionMemory() *ReflexionMemory {
	return &ReflexionMemory{
		Experiences:     make([]Experience, 0),
		SuccessPatterns: make([]SuccessPattern, 0),
		FailurePatterns: make([]FailurePattern, 0),
		Lessons:         make([]Lesson, 0),
	}
}

// Execute runs the Reflexion agent with self-improvement loop
func (a *ReflexionAgent) Execute(ctx *TaskContext, input string, mcpManager *MCPManager, skillRegistry *SkillRegistry) (string, error) {
	startTime := time.Now()

	// Get relevant lessons from memory
	relevantLessons := a.memory.GetRelevantLessons(input)
	enhancedInput := a.enhanceWithLessons(input, relevantLessons)

	// Initial execution
	output, err := a.CallLLM(enhancedInput)
	if err != nil {
		a.recordExperience(ctx.ID, input, "", false, err.Error(), time.Since(startTime).Seconds())
		return "", err
	}

	// Self-improvement loop
	for i := 0; i < a.maxReflections; i++ {
		// Reflect on the output
		reflection, err := a.reflect(input, output)
		if err != nil {
			break // If reflection fails, use current output
		}

		// Check if we've converged (no more improvements needed)
		if reflection.Converged {
			break
		}

		// Apply improvements
		output = reflection.RevisedOutput

		// Record the lesson learned
		if reflection.Reflection != "" {
			a.memory.AddLesson(Lesson{
				ID:          fmt.Sprintf("lesson-%s-%d", ctx.ID, i),
				Description: reflection.Reflection,
				Context:     input,
				Action:      strings.Join(reflection.Improvements, "; "),
				Outcome:     "improved",
				LearnedAt:   time.Now(),
			})
		}
	}

	// Record successful experience
	a.recordExperience(ctx.ID, input, output, true, "", time.Since(startTime).Seconds())

	return output, nil
}

// reflect performs self-reflection on the output
func (a *ReflexionAgent) reflect(input, output string) (*ReflexionResult, error) {
	reflectionPrompt := fmt.Sprintf(`Given the following task and output, perform self-reflection:

TASK: %s

OUTPUT: %s

Analyze the output and provide improvements in JSON format:
{
  "evaluation": {"correct": true/false, "complete": true/false, "issues": []},
  "reflection": "what could be improved",
  "improvements": ["specific improvements"],
  "revised_output": "improved version or same if no changes needed",
  "converged": true/false
}`, input, output)

	response, err := a.CallLLM(reflectionPrompt)
	if err != nil {
		return nil, err
	}

	// Parse the reflection result
	result := &ReflexionResult{
		OriginalOutput: output,
	}

	// Try to parse JSON from response
	if err := json.Unmarshal([]byte(extractJSON(response)), result); err != nil {
		// If parsing fails, assume converged
		result.Converged = true
		result.RevisedOutput = output
	}

	return result, nil
}

// enhanceWithLessons adds relevant lessons to the input
func (a *ReflexionAgent) enhanceWithLessons(input string, lessons []Lesson) string {
	if len(lessons) == 0 {
		return input
	}

	var sb strings.Builder
	sb.WriteString(input)
	sb.WriteString("\n\n--- LESSONS FROM PAST EXPERIENCE ---\n")

	for i, lesson := range lessons {
		sb.WriteString(fmt.Sprintf("%d. %s\n   Action: %s\n", i+1, lesson.Description, lesson.Action))
	}

	return sb.String()
}

// recordExperience records an execution experience
func (a *ReflexionAgent) recordExperience(taskID, input, output string, success bool, errMsg string, duration float64) {
	a.memory.AddExperience(Experience{
		TaskID:    taskID,
		Input:     input,
		Output:    output,
		Success:   success,
		Error:     errMsg,
		Duration:  duration,
		Timestamp: time.Now(),
	})

	// Update patterns
	if success {
		a.memory.UpdateSuccessPattern(input)
	} else {
		a.memory.UpdateFailurePattern(input, errMsg)
	}
}

// AddExperience adds an experience to memory
func (m *ReflexionMemory) AddExperience(exp Experience) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Experiences = append(m.Experiences, exp)

	// Keep only last 1000 experiences
	if len(m.Experiences) > 1000 {
		m.Experiences = m.Experiences[len(m.Experiences)-1000:]
	}
}

// AddLesson adds a lesson to memory
func (m *ReflexionMemory) AddLesson(lesson Lesson) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate lessons
	for i, existing := range m.Lessons {
		if strings.Contains(existing.Description, lesson.Description) ||
			strings.Contains(lesson.Description, existing.Description) {
			m.Lessons[i].AppliedCount++
			return
		}
	}

	m.Lessons = append(m.Lessons, lesson)

	// Keep only last 100 lessons
	if len(m.Lessons) > 100 {
		m.Lessons = m.Lessons[len(m.Lessons)-100:]
	}
}

// GetRelevantLessons retrieves lessons relevant to the given input
func (m *ReflexionMemory) GetRelevantLessons(input string) []Lesson {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inputLower := strings.ToLower(input)
	relevant := make([]Lesson, 0)

	for _, lesson := range m.Lessons {
		contextLower := strings.ToLower(lesson.Context)
		descLower := strings.ToLower(lesson.Description)

		// Check for keyword overlap
		inputWords := strings.Fields(inputLower)
		for _, word := range inputWords {
			if len(word) > 3 && (strings.Contains(contextLower, word) || strings.Contains(descLower, word)) {
				relevant = append(relevant, lesson)
				break
			}
		}
	}

	// Return top 5 most relevant
	if len(relevant) > 5 {
		relevant = relevant[len(relevant)-5:]
	}

	return relevant
}

// UpdateSuccessPattern updates success patterns based on successful execution
func (m *ReflexionMemory) UpdateSuccessPattern(input string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	keywords := extractKeywordsSimple(input)
	pattern := strings.Join(keywords, " ")

	// Find existing pattern or create new
	found := false
	for i, sp := range m.SuccessPatterns {
		if sp.Pattern == pattern {
			m.SuccessPatterns[i].Count++
			m.SuccessPatterns[i].SuccessRate = float64(m.SuccessPatterns[i].Count) / float64(m.SuccessPatterns[i].Count+1)
			found = true
			break
		}
	}

	if !found {
		m.SuccessPatterns = append(m.SuccessPatterns, SuccessPattern{
			Pattern:     pattern,
			Contexts:    []string{input},
			SuccessRate: 1.0,
			Count:       1,
		})
	}
}

// UpdateFailurePattern updates failure patterns based on failed execution
func (m *ReflexionMemory) UpdateFailurePattern(input, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	keywords := extractKeywordsSimple(input)
	pattern := strings.Join(keywords, " ")

	// Find existing pattern or create new
	found := false
	for i, fp := range m.FailurePatterns {
		if fp.Pattern == pattern {
			m.FailurePatterns[i].FailCount++
			if !containsString(fp.Causes, errorMsg) {
				m.FailurePatterns[i].Causes = append(m.FailurePatterns[i].Causes, errorMsg)
			}
			found = true
			break
		}
	}

	if !found {
		m.FailurePatterns = append(m.FailurePatterns, FailurePattern{
			Pattern:   pattern,
			Causes:    []string{errorMsg},
			Solutions: []string{},
			FailCount: 1,
		})
	}
}

// Helper functions
func extractKeywordsSimple(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	keywords := make([]string, 0)
	stopWords := map[string]bool{"the": true, "a": true, "an": true, "and": true, "or": true, "but": true}

	for _, word := range words {
		if len(word) > 3 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	if len(keywords) > 5 {
		keywords = keywords[:5]
	}
	return keywords
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func extractJSON(text string) string {
	// Find JSON object in text
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return "{}"
}

// HumanInLoopManager manages human-in-loop checkpoints
type HumanInLoopManager struct {
	pendingApprovals map[string]*ApprovalRequest
	approvalChan     map[string]chan ApprovalResponse
	mu               sync.RWMutex
	timeout          time.Duration
}

// ApprovalRequest represents a request for human approval
type ApprovalRequest struct {
	ID          string                 `json:"id"`
	TaskID      string                 `json:"task_id"`
	Type        string                 `json:"type"` // file_write, command_execute, api_call
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	FilePath    string                 `json:"file_path,omitempty"`
	Content     string                 `json:"content,omitempty"`
	Diff        string                 `json:"diff,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	Status      string                 `json:"status"` // pending, approved, rejected, timeout
}

// ApprovalResponse represents a response to an approval request
type ApprovalResponse struct {
	RequestID string `json:"request_id"`
	Approved  bool   `json:"approved"`
	Comment   string `json:"comment,omitempty"`
	Modified  string `json:"modified,omitempty"` // Modified content if user edited
}

// NewHumanInLoopManager creates a new human-in-loop manager
func NewHumanInLoopManager(timeout time.Duration) *HumanInLoopManager {
	return &HumanInLoopManager{
		pendingApprovals: make(map[string]*ApprovalRequest),
		approvalChan:     make(map[string]chan ApprovalResponse),
		timeout:          timeout,
	}
}

// RequestApproval requests human approval for an action
func (h *HumanInLoopManager) RequestApproval(ctx context.Context, req *ApprovalRequest) (*ApprovalResponse, error) {
	h.mu.Lock()
	req.CreatedAt = time.Now()
	req.Status = "pending"
	h.pendingApprovals[req.ID] = req
	respChan := make(chan ApprovalResponse, 1)
	h.approvalChan[req.ID] = respChan
	h.mu.Unlock()

	// Wait for approval with timeout
	select {
	case response := <-respChan:
		h.mu.Lock()
		delete(h.pendingApprovals, req.ID)
		delete(h.approvalChan, req.ID)
		h.mu.Unlock()
		return &response, nil
	case <-time.After(h.timeout):
		h.mu.Lock()
		req.Status = "timeout"
		delete(h.pendingApprovals, req.ID)
		delete(h.approvalChan, req.ID)
		h.mu.Unlock()
		return nil, fmt.Errorf("approval request timed out")
	case <-ctx.Done():
		h.mu.Lock()
		delete(h.pendingApprovals, req.ID)
		delete(h.approvalChan, req.ID)
		h.mu.Unlock()
		return nil, ctx.Err()
	}
}

// SubmitApproval submits an approval response
func (h *HumanInLoopManager) SubmitApproval(response ApprovalResponse) error {
	h.mu.RLock()
	respChan, ok := h.approvalChan[response.RequestID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("approval request %s not found or expired", response.RequestID)
	}

	respChan <- response
	return nil
}

// GetPendingApprovals returns all pending approval requests
func (h *HumanInLoopManager) GetPendingApprovals() []*ApprovalRequest {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*ApprovalRequest, 0, len(h.pendingApprovals))
	for _, req := range h.pendingApprovals {
		result = append(result, req)
	}
	return result
}

// GetPendingApproval returns a specific pending approval request
func (h *HumanInLoopManager) GetPendingApproval(id string) (*ApprovalRequest, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	req, ok := h.pendingApprovals[id]
	return req, ok
}

// AutoApprove automatically approves requests matching certain criteria
func (h *HumanInLoopManager) AutoApprove(requestType string, filePatterns []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, req := range h.pendingApprovals {
		if req.Type != requestType {
			continue
		}

		// Check if file matches any pattern
		for _, pattern := range filePatterns {
			if matchPattern(req.FilePath, pattern) {
				if respChan, ok := h.approvalChan[id]; ok {
					respChan <- ApprovalResponse{
						RequestID: id,
						Approved:  true,
						Comment:   "auto-approved by pattern match",
					}
				}
				break
			}
		}
	}
}

func matchPattern(filepath, pattern string) bool {
	// Simple pattern matching with wildcards
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		ext := pattern[1:]
		return strings.HasSuffix(filepath, ext)
	}
	return strings.Contains(filepath, pattern)
}
