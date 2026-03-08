package agentsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MCPToolExecutor interface for MCP tool execution (allows dependency injection)
type MCPToolExecutor interface {
	CallTool(serverName string, toolName string, args map[string]interface{}) (string, error)
}

// WorkflowEngine manages workflow execution with DAG-based orchestration
type WorkflowEngine struct {
	db           *gorm.DB
	agentService *CustomAgentService
	mcpExecutor  MCPToolExecutor
	runningJobs  map[string]*workflowJob
	mu           sync.RWMutex
}

type workflowJob struct {
	run       *models.WorkflowRun
	workflow  *models.Workflow
	steps     map[string]*models.WorkflowStep
	edges     map[string][]*models.WorkflowEdge // sourceID -> edges
	ctx       context.Context
	cancel    context.CancelFunc
	eventChan chan WorkflowEvent
}

// WorkflowEvent represents a workflow execution event
type WorkflowEvent struct {
	Type      string      `json:"type"` // step_started, step_completed, step_failed, workflow_paused, workflow_completed
	StepID    string      `json:"step_id,omitempty"`
	StepName  string      `json:"step_name,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(db *gorm.DB, agentService *CustomAgentService, mcpExecutor ...MCPToolExecutor) *WorkflowEngine {
	engine := &WorkflowEngine{
		db:           db,
		agentService: agentService,
		runningJobs:  make(map[string]*workflowJob),
	}
	if len(mcpExecutor) > 0 && mcpExecutor[0] != nil {
		engine.mcpExecutor = mcpExecutor[0]
	}
	return engine
}

// CreateWorkflow creates a new workflow
func (e *WorkflowEngine) CreateWorkflow(ctx context.Context, userID uint, workflow *models.Workflow) (*models.Workflow, error) {
	workflow.ID = uuid.New().String()
	workflow.UserID = userID
	workflow.Status = "draft"
	workflow.Version = "1.0.0"

	// Initialize metrics
	metrics := map[string]interface{}{
		"successRate": 0.0,
		"avgDuration": 0,
		"totalRuns":   0,
	}
	metricsJSON, _ := json.Marshal(metrics)
	workflow.Metrics = metricsJSON

	if err := e.db.WithContext(ctx).Create(workflow).Error; err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return workflow, nil
}

// GetWorkflow retrieves a workflow with all steps and edges
func (e *WorkflowEngine) GetWorkflow(ctx context.Context, id string, userID uint) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := e.db.WithContext(ctx).
		Preload("Steps").
		Preload("Edges").
		Where("id = ? AND (user_id = ? OR is_public = true)", id, userID).
		First(&workflow).Error; err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}
	return &workflow, nil
}

// ListWorkflows lists all workflows for a user
func (e *WorkflowEngine) ListWorkflows(ctx context.Context, userID uint, includePublic bool) ([]models.Workflow, error) {
	var workflows []models.Workflow
	query := e.db.WithContext(ctx)

	if includePublic {
		query = query.Where("user_id = ? OR is_public = true", userID)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Order("updated_at DESC").Find(&workflows).Error; err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	return workflows, nil
}

// UpdateWorkflow updates workflow metadata
func (e *WorkflowEngine) UpdateWorkflow(ctx context.Context, id string, userID uint, updates map[string]interface{}) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := e.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&workflow).Error; err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	if err := e.db.WithContext(ctx).Model(&workflow).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	return &workflow, nil
}

// DeleteWorkflow soft deletes a workflow
func (e *WorkflowEngine) DeleteWorkflow(ctx context.Context, id string, userID uint) error {
	result := e.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.Workflow{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete workflow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("workflow not found or unauthorized")
	}
	return nil
}

// AddStep adds a step to a workflow
func (e *WorkflowEngine) AddStep(ctx context.Context, workflowID string, userID uint, step *models.WorkflowStep) (*models.WorkflowStep, error) {
	// Verify workflow ownership
	var workflow models.Workflow
	if err := e.db.WithContext(ctx).Where("id = ? AND user_id = ?", workflowID, userID).First(&workflow).Error; err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	step.ID = uuid.New().String()
	step.WorkflowID = workflowID

	if err := e.db.WithContext(ctx).Create(step).Error; err != nil {
		return nil, fmt.Errorf("failed to add step: %w", err)
	}

	return step, nil
}

// UpdateStep updates a workflow step
func (e *WorkflowEngine) UpdateStep(ctx context.Context, stepID string, userID uint, updates map[string]interface{}) (*models.WorkflowStep, error) {
	var step models.WorkflowStep
	if err := e.db.WithContext(ctx).
		Joins("JOIN workflows ON workflow_steps.workflow_id = workflows.id").
		Where("workflow_steps.id = ? AND workflows.user_id = ?", stepID, userID).
		First(&step).Error; err != nil {
		return nil, fmt.Errorf("step not found: %w", err)
	}

	if err := e.db.WithContext(ctx).Model(&step).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	return &step, nil
}

// DeleteStep deletes a workflow step
func (e *WorkflowEngine) DeleteStep(ctx context.Context, stepID string, userID uint) error {
	// First verify ownership through workflow
	var step models.WorkflowStep
	if err := e.db.WithContext(ctx).
		Joins("JOIN workflows ON workflow_steps.workflow_id = workflows.id").
		Where("workflow_steps.id = ? AND workflows.user_id = ?", stepID, userID).
		First(&step).Error; err != nil {
		return fmt.Errorf("step not found: %w", err)
	}

	// Delete associated edges
	e.db.WithContext(ctx).Where("source_step_id = ? OR target_step_id = ?", stepID, stepID).Delete(&models.WorkflowEdge{})

	// Delete step
	if err := e.db.WithContext(ctx).Delete(&step).Error; err != nil {
		return fmt.Errorf("failed to delete step: %w", err)
	}

	return nil
}

// AddEdge adds an edge between steps
func (e *WorkflowEngine) AddEdge(ctx context.Context, workflowID string, userID uint, edge *models.WorkflowEdge) (*models.WorkflowEdge, error) {
	// Verify workflow ownership
	var workflow models.Workflow
	if err := e.db.WithContext(ctx).Where("id = ? AND user_id = ?", workflowID, userID).First(&workflow).Error; err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	edge.ID = uuid.New().String()
	edge.WorkflowID = workflowID

	if err := e.db.WithContext(ctx).Create(edge).Error; err != nil {
		return nil, fmt.Errorf("failed to add edge: %w", err)
	}

	return edge, nil
}

// DeleteEdge deletes an edge
func (e *WorkflowEngine) DeleteEdge(ctx context.Context, edgeID string, userID uint) error {
	var edge models.WorkflowEdge
	if err := e.db.WithContext(ctx).
		Joins("JOIN workflows ON workflow_edges.workflow_id = workflows.id").
		Where("workflow_edges.id = ? AND workflows.user_id = ?", edgeID, userID).
		First(&edge).Error; err != nil {
		return fmt.Errorf("edge not found: %w", err)
	}

	if err := e.db.WithContext(ctx).Delete(&edge).Error; err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	return nil
}

// StartWorkflow begins executing a workflow
func (e *WorkflowEngine) StartWorkflow(ctx context.Context, workflowID string, userID uint, input map[string]interface{}) (*models.WorkflowRun, <-chan WorkflowEvent, error) {
	workflow, err := e.GetWorkflow(ctx, workflowID, userID)
	if err != nil {
		return nil, nil, err
	}

	if len(workflow.Steps) == 0 {
		return nil, nil, fmt.Errorf("workflow has no steps")
	}

	inputJSON, _ := json.Marshal(input)
	variablesJSON, _ := json.Marshal(map[string]interface{}{
		"input": input,
	})

	run := &models.WorkflowRun{
		ID:          uuid.New().String(),
		WorkflowID:  workflowID,
		UserID:      userID,
		Status:      "running",
		TriggerType: "manual",
		Input:       inputJSON,
		Variables:   variablesJSON,
		StartedAt:   time.Now(),
		CreatedAt:   time.Now(),
	}

	if err := e.db.WithContext(ctx).Create(run).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create workflow run: %w", err)
	}

	// Build step and edge maps
	steps := make(map[string]*models.WorkflowStep)
	for i := range workflow.Steps {
		steps[workflow.Steps[i].ID] = &workflow.Steps[i]
	}

	edges := make(map[string][]*models.WorkflowEdge)
	for i := range workflow.Edges {
		edge := &workflow.Edges[i]
		edges[edge.SourceStepID] = append(edges[edge.SourceStepID], edge)
	}

	jobCtx, cancel := context.WithCancel(context.Background())
	eventChan := make(chan WorkflowEvent, 100)

	job := &workflowJob{
		run:       run,
		workflow:  workflow,
		steps:     steps,
		edges:     edges,
		ctx:       jobCtx,
		cancel:    cancel,
		eventChan: eventChan,
	}

	e.mu.Lock()
	e.runningJobs[run.ID] = job
	e.mu.Unlock()

	// Start execution in goroutine
	go e.executeWorkflow(job)

	return run, eventChan, nil
}

// executeWorkflow runs the workflow DAG
func (e *WorkflowEngine) executeWorkflow(job *workflowJob) {
	defer func() {
		close(job.eventChan)
		e.mu.Lock()
		delete(e.runningJobs, job.run.ID)
		e.mu.Unlock()
	}()

	// Find entry points (steps with no incoming edges)
	entryPoints := e.findEntryPoints(job)
	if len(entryPoints) == 0 {
		e.failWorkflow(job, "no entry point found in workflow")
		return
	}

	// Execute steps using topological traversal
	completed := make(map[string]bool)
	stepResults := make(map[string]interface{})

	var executeStep func(stepID string) error
	executeStep = func(stepID string) error {
		if completed[stepID] {
			return nil
		}

		step := job.steps[stepID]
		if step == nil {
			return fmt.Errorf("step not found: %s", stepID)
		}

		// Check if paused at checkpoint
		if step.Type == "checkpoint" {
			e.pauseAtCheckpoint(job, step)
			return nil
		}

		// Send step started event
		job.eventChan <- WorkflowEvent{
			Type:      "step_started",
			StepID:    stepID,
			StepName:  step.Name,
			Timestamp: time.Now(),
		}

		// Execute based on step type
		var result interface{}
		var execErr error

		switch step.Type {
		case "agent":
			result, execErr = e.executeAgentStep(job, step, stepResults)
		case "condition":
			result, execErr = e.executeConditionStep(job, step, stepResults)
		case "transform":
			result, execErr = e.executeTransformStep(job, step, stepResults)
		case "tool":
			result, execErr = e.executeToolStep(job, step, stepResults)
		default:
			execErr = fmt.Errorf("unknown step type: %s", step.Type)
		}

		// Record step run
		stepRun := &models.WorkflowStepRun{
			ID:            uuid.New().String(),
			WorkflowRunID: job.run.ID,
			StepID:        stepID,
			StartedAt:     time.Now(),
		}

		now := time.Now()
		if execErr != nil {
			stepRun.Status = "failed"
			stepRun.ErrorMessage = execErr.Error()
			stepRun.CompletedAt = &now

			job.eventChan <- WorkflowEvent{
				Type:      "step_failed",
				StepID:    stepID,
				StepName:  step.Name,
				Data:      map[string]string{"error": execErr.Error()},
				Timestamp: time.Now(),
			}

			return execErr
		}

		stepRun.Status = "completed"
		resultJSON, _ := json.Marshal(result)
		stepRun.Output = resultJSON
		stepRun.CompletedAt = &now
		e.db.Create(stepRun)

		completed[stepID] = true
		stepResults[stepID] = result

		job.eventChan <- WorkflowEvent{
			Type:      "step_completed",
			StepID:    stepID,
			StepName:  step.Name,
			Data:      result,
			Timestamp: time.Now(),
		}

		// Find and execute next steps
		for _, edge := range job.edges[stepID] {
			// Check edge condition
			if e.evaluateEdgeCondition(edge, result) {
				if err := executeStep(edge.TargetStepID); err != nil {
					return err
				}
			}
		}

		return nil
	}

	// Execute from all entry points
	for _, entryID := range entryPoints {
		select {
		case <-job.ctx.Done():
			return
		default:
			if err := executeStep(entryID); err != nil {
				e.failWorkflow(job, err.Error())
				return
			}
		}
	}

	// Workflow completed successfully
	e.completeWorkflow(job, stepResults)
}

func (e *WorkflowEngine) findEntryPoints(job *workflowJob) []string {
	hasIncoming := make(map[string]bool)
	for _, edgeList := range job.edges {
		for _, edge := range edgeList {
			hasIncoming[edge.TargetStepID] = true
		}
	}

	var entryPoints []string
	for stepID := range job.steps {
		if !hasIncoming[stepID] {
			entryPoints = append(entryPoints, stepID)
		}
	}
	return entryPoints
}

func (e *WorkflowEngine) executeAgentStep(job *workflowJob, step *models.WorkflowStep, prevResults map[string]interface{}) (interface{}, error) {
	if step.AgentID == nil {
		return nil, fmt.Errorf("agent step missing agent_id")
	}

	// Build input from mapping
	input := e.mapInputs(step.InputMapping, job.run.Variables, prevResults)
	inputStr, _ := json.Marshal(input)

	execution, err := e.agentService.ExecuteAgent(job.ctx, *step.AgentID, job.run.UserID, string(inputStr), "")
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"output":  execution.Output,
		"latency": execution.LatencyMs,
		"tokens":  execution.TokensUsed,
	}, nil
}

func (e *WorkflowEngine) executeConditionStep(job *workflowJob, step *models.WorkflowStep, prevResults map[string]interface{}) (interface{}, error) {
	var condition map[string]interface{}
	json.Unmarshal(step.Condition, &condition)

	// Simple condition evaluation
	field, _ := condition["field"].(string)
	operator, _ := condition["operator"].(string)
	value := condition["value"]

	// Get field value from previous results
	var fieldValue interface{}
	for _, result := range prevResults {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if v, exists := resultMap[field]; exists {
				fieldValue = v
				break
			}
		}
	}

	var result bool
	switch operator {
	case "eq", "==":
		result = fieldValue == value
	case "ne", "!=":
		result = fieldValue != value
	case "gt", ">":
		if fv, ok := fieldValue.(float64); ok {
			if v, ok := value.(float64); ok {
				result = fv > v
			}
		}
	case "lt", "<":
		if fv, ok := fieldValue.(float64); ok {
			if v, ok := value.(float64); ok {
				result = fv < v
			}
		}
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if substr, ok := value.(string); ok {
				result = len(str) > 0 && len(substr) > 0 && (str == substr || len(str) > len(substr))
			}
		}
	default:
		result = false
	}

	return map[string]interface{}{
		"condition": result,
		"field":     field,
		"operator":  operator,
		"value":     value,
	}, nil
}

func (e *WorkflowEngine) executeTransformStep(job *workflowJob, step *models.WorkflowStep, prevResults map[string]interface{}) (interface{}, error) {
	// Apply input/output mapping transformation
	return e.mapInputs(step.OutputMapping, job.run.Variables, prevResults), nil
}

func (e *WorkflowEngine) executeToolStep(job *workflowJob, step *models.WorkflowStep, prevResults map[string]interface{}) (interface{}, error) {
	// Check if MCP executor is available
	if e.mcpExecutor == nil {
		toolIDStr := ""
		if step.ToolID != nil {
			toolIDStr = *step.ToolID
		}
		return map[string]interface{}{
			"status": "error",
			"error":  "MCP executor not configured",
			"tool":   toolIDStr,
		}, fmt.Errorf("MCP executor not available")
	}

	// Get tool ID
	if step.ToolID == nil || *step.ToolID == "" {
		return nil, fmt.Errorf("tool ID not specified for step")
	}
	toolID := *step.ToolID

	// Parse tool ID: format is "server/tool" or just "tool" (uses default server)
	serverName := "filesystem-local" // Default server
	toolName := toolID

	if idx := strings.Index(toolID, "/"); idx > 0 {
		serverName = toolID[:idx]
		toolName = toolID[idx+1:]
	}

	// Prepare arguments from input mapping
	args := e.mapInputs(step.InputMapping, job.run.Variables, prevResults)

	// Execute the tool via MCP
	result, err := e.mcpExecutor.CallTool(serverName, toolName, args)
	if err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
			"tool":   toolID,
		}, err
	}

	// Try to parse result as JSON
	var parsedResult interface{}
	if err := json.Unmarshal([]byte(result), &parsedResult); err != nil {
		// If not JSON, return as string
		parsedResult = result
	}

	return map[string]interface{}{
		"status": "success",
		"tool":   toolID,
		"output": parsedResult,
	}, nil
}

func (e *WorkflowEngine) mapInputs(mapping models.JSON, variables models.JSON, prevResults map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	var mappingMap map[string]interface{}
	json.Unmarshal(mapping, &mappingMap)

	var varsMap map[string]interface{}
	json.Unmarshal(variables, &varsMap)

	for key, source := range mappingMap {
		if sourceStr, ok := source.(string); ok {
			// Check if it's a reference like "step.output" or "variables.input"
			if len(sourceStr) > 0 && sourceStr[0] == '$' {
				// Variable reference
				parts := splitPath(sourceStr[1:])
				if len(parts) >= 2 {
					switch parts[0] {
					case "step":
						if stepResult, ok := prevResults[parts[1]]; ok {
							result[key] = getNestedValue(stepResult, parts[2:])
						}
					case "var", "variables":
						result[key] = getNestedValue(varsMap, parts[1:])
					}
				}
			} else {
				result[key] = source
			}
		} else {
			result[key] = source
		}
	}

	return result
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, ch := range path {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func getNestedValue(obj interface{}, path []string) interface{} {
	if len(path) == 0 {
		return obj
	}
	if m, ok := obj.(map[string]interface{}); ok {
		return getNestedValue(m[path[0]], path[1:])
	}
	return nil
}

func (e *WorkflowEngine) evaluateEdgeCondition(edge *models.WorkflowEdge, result interface{}) bool {
	if edge.Label == "" || edge.Label == "success" {
		return true
	}

	if resultMap, ok := result.(map[string]interface{}); ok {
		if condition, exists := resultMap["condition"]; exists {
			if edge.Label == "true" {
				return condition == true
			}
			if edge.Label == "false" {
				return condition == false
			}
		}
	}

	return edge.Label == "success"
}

func (e *WorkflowEngine) pauseAtCheckpoint(job *workflowJob, step *models.WorkflowStep) {
	now := time.Now()
	job.run.Status = "paused"
	job.run.CurrentStepID = &step.ID
	job.run.PausedAt = &now

	checkpointData := map[string]interface{}{
		"step_id":   step.ID,
		"step_name": step.Name,
		"message":   "Waiting for human approval",
	}
	checkpointJSON, _ := json.Marshal(checkpointData)
	job.run.CheckpointData = checkpointJSON

	e.db.Save(job.run)

	job.eventChan <- WorkflowEvent{
		Type:      "workflow_paused",
		StepID:    step.ID,
		StepName:  step.Name,
		Data:      checkpointData,
		Timestamp: time.Now(),
	}
}

func (e *WorkflowEngine) failWorkflow(job *workflowJob, message string) {
	now := time.Now()
	job.run.Status = "failed"
	job.run.ErrorMessage = message
	job.run.CompletedAt = &now
	job.run.DurationMs = int(now.Sub(job.run.StartedAt).Milliseconds())
	e.db.Save(job.run)

	job.eventChan <- WorkflowEvent{
		Type:      "workflow_failed",
		Data:      map[string]string{"error": message},
		Timestamp: time.Now(),
	}
}

func (e *WorkflowEngine) completeWorkflow(job *workflowJob, results map[string]interface{}) {
	now := time.Now()
	job.run.Status = "completed"
	job.run.CompletedAt = &now
	job.run.DurationMs = int(now.Sub(job.run.StartedAt).Milliseconds())

	outputJSON, _ := json.Marshal(results)
	job.run.Output = outputJSON

	e.db.Save(job.run)

	// Update workflow metrics
	go e.updateWorkflowMetrics(job.workflow.ID, job.run)

	job.eventChan <- WorkflowEvent{
		Type:      "workflow_completed",
		Data:      results,
		Timestamp: time.Now(),
	}
}

func (e *WorkflowEngine) updateWorkflowMetrics(workflowID string, run *models.WorkflowRun) {
	var workflow models.Workflow
	if err := e.db.First(&workflow, "id = ?", workflowID).Error; err != nil {
		return
	}

	var metrics map[string]interface{}
	json.Unmarshal(workflow.Metrics, &metrics)

	totalRuns := metrics["totalRuns"].(float64) + 1
	successRate := metrics["successRate"].(float64)
	avgDuration := metrics["avgDuration"].(float64)

	if run.Status == "completed" {
		successRate = (successRate*(totalRuns-1) + 1) / totalRuns
	} else {
		successRate = (successRate * (totalRuns - 1)) / totalRuns
	}

	avgDuration = (avgDuration*(totalRuns-1) + float64(run.DurationMs)) / totalRuns

	metrics["totalRuns"] = totalRuns
	metrics["successRate"] = successRate
	metrics["avgDuration"] = avgDuration

	metricsJSON, _ := json.Marshal(metrics)
	e.db.Model(&workflow).Update("metrics", metricsJSON)
}

// ResumeWorkflow resumes a paused workflow
func (e *WorkflowEngine) ResumeWorkflow(ctx context.Context, runID string, userID uint, approved bool) error {
	var run models.WorkflowRun
	if err := e.db.WithContext(ctx).Where("id = ? AND user_id = ? AND status = ?", runID, userID, "paused").First(&run).Error; err != nil {
		return fmt.Errorf("paused workflow not found: %w", err)
	}

	if !approved {
		run.Status = "cancelled"
		now := time.Now()
		run.CompletedAt = &now
		return e.db.Save(&run).Error
	}

	// Resume execution
	run.Status = "running"
	run.PausedAt = nil
	e.db.Save(&run)

	// Would need to restart execution from checkpoint
	// This is a simplified implementation
	return nil
}

// CancelWorkflow cancels a running workflow
func (e *WorkflowEngine) CancelWorkflow(ctx context.Context, runID string, userID uint) error {
	e.mu.RLock()
	job, exists := e.runningJobs[runID]
	e.mu.RUnlock()

	if exists {
		job.cancel()
	}

	var run models.WorkflowRun
	if err := e.db.WithContext(ctx).Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
		return fmt.Errorf("workflow run not found: %w", err)
	}

	run.Status = "cancelled"
	now := time.Now()
	run.CompletedAt = &now
	return e.db.Save(&run).Error
}

// GetWorkflowRun retrieves a workflow run
func (e *WorkflowEngine) GetWorkflowRun(ctx context.Context, runID string, userID uint) (*models.WorkflowRun, error) {
	var run models.WorkflowRun
	if err := e.db.WithContext(ctx).Where("id = ? AND user_id = ?", runID, userID).First(&run).Error; err != nil {
		return nil, fmt.Errorf("workflow run not found: %w", err)
	}
	return &run, nil
}

// GetWorkflowRunHistory returns run history for a workflow
func (e *WorkflowEngine) GetWorkflowRunHistory(ctx context.Context, workflowID string, userID uint, limit int) ([]models.WorkflowRun, error) {
	var runs []models.WorkflowRun
	if err := e.db.WithContext(ctx).
		Where("workflow_id = ? AND user_id = ?", workflowID, userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("failed to get run history: %w", err)
	}
	return runs, nil
}

// ExportWorkflow exports workflow configuration
func (e *WorkflowEngine) ExportWorkflow(ctx context.Context, id string, userID uint) (map[string]interface{}, error) {
	workflow, err := e.GetWorkflow(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	export := map[string]interface{}{
		"manifest": "1.0",
		"workflow": map[string]interface{}{
			"name":        workflow.Name,
			"description": workflow.Description,
			"version":     workflow.Version,
			"icon":        workflow.Icon,
		},
		"steps": workflow.Steps,
		"edges": workflow.Edges,
	}

	return export, nil
}

// ImportWorkflow imports a workflow from exported configuration
func (e *WorkflowEngine) ImportWorkflow(ctx context.Context, userID uint, data map[string]interface{}) (*models.Workflow, error) {
	workflowData, ok := data["workflow"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid import format: missing workflow data")
	}

	workflow := &models.Workflow{
		UserID: userID,
	}

	if name, ok := workflowData["name"].(string); ok {
		workflow.Name = name + " (Imported)"
	}
	if desc, ok := workflowData["description"].(string); ok {
		workflow.Description = desc
	}
	if icon, ok := workflowData["icon"].(string); ok {
		workflow.Icon = icon
	}

	newWorkflow, err := e.CreateWorkflow(ctx, userID, workflow)
	if err != nil {
		return nil, err
	}

	// Import steps
	if steps, ok := data["steps"].([]interface{}); ok {
		for _, stepData := range steps {
			if stepMap, ok := stepData.(map[string]interface{}); ok {
				step := &models.WorkflowStep{
					WorkflowID: newWorkflow.ID,
				}
				if name, ok := stepMap["name"].(string); ok {
					step.Name = name
				}
				if stepType, ok := stepMap["type"].(string); ok {
					step.Type = stepType
				}
				// Add more field mappings as needed
				e.AddStep(ctx, newWorkflow.ID, userID, step)
			}
		}
	}

	return newWorkflow, nil
}
