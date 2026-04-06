package agentsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend/internal/models"
	"backend/internal/pkg/llm"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomAgentService manages custom agent CRUD and execution
type CustomAgentService struct {
	db        *gorm.DB
	llmClient llm.Client
}

// NewCustomAgentService creates a new custom agent service
func NewCustomAgentService(db *gorm.DB, llmClient llm.Client) *CustomAgentService {
	return &CustomAgentService{
		db:        db,
		llmClient: llmClient,
	}
}

// AgentTemplate represents predefined cognitive architecture templates
type AgentTemplate struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Architecture   string                 `json:"architecture"`
	PromptTemplate map[string]interface{} `json:"prompt_template"`
	DefaultPersona map[string]interface{} `json:"default_persona"`
}

// GetTemplates returns predefined agent templates
func (s *CustomAgentService) GetTemplates() []AgentTemplate {
	return []AgentTemplate{
		{
			ID:           "react",
			Name:         "ReAct Agent",
			Description:  "Reasoning and Acting - thinks step by step then acts",
			Architecture: "react",
			PromptTemplate: map[string]interface{}{
				"system": `You are an AI assistant that uses the ReAct framework.
For each task:
1. Thought: Analyze what needs to be done
2. Action: Choose an action to take
3. Observation: Observe the result
4. Repeat until task is complete
5. Final Answer: Provide the final response`,
			},
			DefaultPersona: map[string]interface{}{
				"style": "analytical",
				"tone":  "professional",
			},
		},
		{
			ID:           "reflexion",
			Name:         "Reflexion Agent",
			Description:  "Self-reflecting agent that learns from mistakes",
			Architecture: "reflexion",
			PromptTemplate: map[string]interface{}{
				"system": `You are a self-improving AI assistant using the Reflexion framework.
After each response:
1. Evaluate: Assess the quality of your response
2. Reflect: Identify what could be improved
3. Refine: Apply improvements to generate better output
4. Learn: Store insights for future tasks`,
			},
			DefaultPersona: map[string]interface{}{
				"style": "thoughtful",
				"tone":  "introspective",
			},
		},
		{
			ID:           "tot",
			Name:         "Tree of Thought",
			Description:  "Explores multiple reasoning paths before deciding",
			Architecture: "tot",
			PromptTemplate: map[string]interface{}{
				"system": `You are an AI assistant using Tree of Thought reasoning.
For complex problems:
1. Generate multiple possible approaches
2. Evaluate each approach's potential
3. Expand promising branches
4. Prune unlikely paths
5. Select the best solution path`,
			},
			DefaultPersona: map[string]interface{}{
				"style": "exploratory",
				"tone":  "methodical",
			},
		},
		{
			ID:           "cot",
			Name:         "Chain of Thought",
			Description:  "Step-by-step reasoning for complex tasks",
			Architecture: "cot",
			PromptTemplate: map[string]interface{}{
				"system": `You are an AI assistant that thinks step by step.
For each problem:
1. Break down the problem into smaller parts
2. Solve each part sequentially
3. Show your reasoning at each step
4. Combine results for final answer`,
			},
			DefaultPersona: map[string]interface{}{
				"style": "logical",
				"tone":  "educational",
			},
		},
		{
			ID:           "coding",
			Name:         "Code Expert",
			Description:  "Specialized agent for coding tasks",
			Architecture: "react",
			PromptTemplate: map[string]interface{}{
				"system": `You are an expert programmer assistant.
You excel at:
- Writing clean, efficient code
- Debugging and fixing issues
- Code review and optimization
- Explaining technical concepts
Always provide well-commented code with explanations.`,
			},
			DefaultPersona: map[string]interface{}{
				"expertise": []string{"programming", "debugging", "architecture"},
				"style":     "technical",
				"tone":      "precise",
			},
		},
		{
			ID:           "analyst",
			Name:         "Data Analyst",
			Description:  "Specialized for data analysis and insights",
			Architecture: "cot",
			PromptTemplate: map[string]interface{}{
				"system": `You are a data analysis expert.
Your approach:
1. Understand the data context
2. Identify patterns and anomalies
3. Apply statistical methods
4. Generate actionable insights
5. Visualize findings when helpful`,
			},
			DefaultPersona: map[string]interface{}{
				"expertise": []string{"statistics", "visualization", "insights"},
				"style":     "analytical",
				"tone":      "data-driven",
			},
		},
	}
}

// CreateAgent creates a new custom agent
func (s *CustomAgentService) CreateAgent(ctx context.Context, userID uint, agent *models.CustomAgent) (*models.CustomAgent, error) {
	agent.ID = uuid.New().String()
	agent.UserID = userID
	agent.Status = "draft"
	agent.Version = "1.0.0"

	// Initialize metrics
	metrics := map[string]interface{}{
		"successRate":     0.0,
		"avgLatency":      0,
		"totalExecutions": 0,
	}
	metricsJSON, _ := json.Marshal(metrics)
	agent.Metrics = metricsJSON

	if err := s.db.WithContext(ctx).Create(agent).Error; err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}

// GetAgent retrieves an agent by ID
func (s *CustomAgentService) GetAgent(ctx context.Context, id string, userID uint) (*models.CustomAgent, error) {
	var agent models.CustomAgent
	if err := s.db.WithContext(ctx).Where("id = ? AND (user_id = ? OR is_public = true)", id, userID).First(&agent).Error; err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}
	return &agent, nil
}

// ListAgents lists all agents for a user
func (s *CustomAgentService) ListAgents(ctx context.Context, userID uint, includePublic bool) ([]models.CustomAgent, error) {
	var agents []models.CustomAgent
	query := s.db.WithContext(ctx)

	if includePublic {
		query = query.Where("user_id = ? OR is_public = true", userID)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Order("updated_at DESC").Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	return agents, nil
}

// UpdateAgent updates an existing agent
func (s *CustomAgentService) UpdateAgent(ctx context.Context, id string, userID uint, updates map[string]interface{}) (*models.CustomAgent, error) {
	var agent models.CustomAgent
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&agent).Error; err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	if err := s.db.WithContext(ctx).Model(&agent).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	return &agent, nil
}

// DeleteAgent soft deletes an agent
func (s *CustomAgentService) DeleteAgent(ctx context.Context, id string, userID uint) error {
	result := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.CustomAgent{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("agent not found or unauthorized")
	}
	return nil
}

// DuplicateAgent creates a copy of an agent
func (s *CustomAgentService) DuplicateAgent(ctx context.Context, id string, userID uint, newName string) (*models.CustomAgent, error) {
	original, err := s.GetAgent(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	newAgent := &models.CustomAgent{
		ID:                    uuid.New().String(),
		UserID:                userID,
		Name:                  newName,
		Description:           original.Description,
		Icon:                  original.Icon,
		CognitiveArchitecture: original.CognitiveArchitecture,
		Status:                "draft",
		Version:               "1.0.0",
		Tags:                  original.Tags,
		PromptTemplate:        original.PromptTemplate,
		Persona:               original.Persona,
		MemoryConfig:          original.MemoryConfig,
		ToolBindings:          original.ToolBindings,
		SelfImproveConfig:     original.SelfImproveConfig,
		ModelPreferences:      original.ModelPreferences,
		InputSchema:           original.InputSchema,
		OutputSchema:          original.OutputSchema,
		ParentAgentID:         &original.ID,
	}

	if err := s.db.WithContext(ctx).Create(newAgent).Error; err != nil {
		return nil, fmt.Errorf("failed to duplicate agent: %w", err)
	}

	return newAgent, nil
}

// ExecuteAgent runs an agent with given input
func (s *CustomAgentService) ExecuteAgent(ctx context.Context, agentID string, userID uint, input string, model string) (*models.AgentExecution, error) {
	agent, err := s.GetAgent(ctx, agentID, userID)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Build prompt from agent configuration
	prompt := s.buildPrompt(agent, input)

	// Use specified model or fallback
	if model == "" {
		var prefs map[string]interface{}
		if err := json.Unmarshal(agent.ModelPreferences, &prefs); err == nil {
			if preferred, ok := prefs["preferredModel"].(string); ok && preferred != "" {
				model = preferred
			}
		}
		if model == "" {
			model = "deepseek-chat"
		}
	}

	// Build full prompt
	fullPrompt := fmt.Sprintf("System: %s\n\nUser: %s", prompt.System, prompt.User)

	// Call LLM
	response, err := s.llmClient.Complete(ctx, fullPrompt, llm.CompletionOptions{
		Model:       model,
		Temperature: 0.7,
	})

	endTime := time.Now()
	latencyMs := int(endTime.Sub(startTime).Milliseconds())

	// Create execution record
	execution := &models.AgentExecution{
		ID:        uuid.New().String(),
		AgentID:   agentID,
		UserID:    userID,
		Input:     input,
		Model:     model,
		LatencyMs: latencyMs,
		CreatedAt: time.Now(),
	}

	if err != nil {
		execution.Status = "error"
		execution.ErrorMessage = err.Error()
	} else {
		execution.Status = "success"
		execution.Output = response
		execution.TokensUsed = s.estimateTokens(prompt.System + prompt.User + response)
	}

	// Save execution
	if saveErr := s.db.WithContext(ctx).Create(execution).Error; saveErr != nil {
		fmt.Printf("Failed to save execution: %v\n", saveErr)
	}

	// Update agent metrics
	go s.updateAgentMetrics(agentID, execution)

	return execution, err
}

type agentPrompt struct {
	System string
	User   string
}

func (s *CustomAgentService) buildPrompt(agent *models.CustomAgent, input string) agentPrompt {
	var template map[string]interface{}
	json.Unmarshal(agent.PromptTemplate, &template)

	var persona map[string]interface{}
	json.Unmarshal(agent.Persona, &persona)

	system := ""
	if sys, ok := template["system"].(string); ok {
		system = sys
	}

	// Add persona context
	if persona != nil {
		if style, ok := persona["style"].(string); ok {
			system += fmt.Sprintf("\n\nCommunication style: %s", style)
		}
		if tone, ok := persona["tone"].(string); ok {
			system += fmt.Sprintf("\nTone: %s", tone)
		}
		if expertise, ok := persona["expertise"].([]interface{}); ok {
			expertiseStrs := make([]string, len(expertise))
			for i, e := range expertise {
				expertiseStrs[i] = fmt.Sprintf("%v", e)
			}
			system += fmt.Sprintf("\nExpertise areas: %v", expertiseStrs)
		}
	}

	user := input
	if userTemplate, ok := template["user"].(string); ok && userTemplate != "" {
		user = fmt.Sprintf("%s\n\nUser input: %s", userTemplate, input)
	}

	return agentPrompt{System: system, User: user}
}

func (s *CustomAgentService) estimateTokens(text string) int {
	// Rough estimate: ~4 chars per token
	return len(text) / 4
}

func (s *CustomAgentService) updateAgentMetrics(agentID string, execution *models.AgentExecution) {
	var agent models.CustomAgent
	if err := s.db.First(&agent, "id = ?", agentID).Error; err != nil {
		return
	}

	var metrics map[string]interface{}
	json.Unmarshal(agent.Metrics, &metrics)

	totalExec := metrics["totalExecutions"].(float64) + 1
	successRate := metrics["successRate"].(float64)
	avgLatency := metrics["avgLatency"].(float64)

	if execution.Status == "success" {
		successRate = (successRate*(totalExec-1) + 1) / totalExec
	} else {
		successRate = (successRate * (totalExec - 1)) / totalExec
	}

	avgLatency = (avgLatency*(totalExec-1) + float64(execution.LatencyMs)) / totalExec

	metrics["totalExecutions"] = totalExec
	metrics["successRate"] = successRate
	metrics["avgLatency"] = avgLatency

	metricsJSON, _ := json.Marshal(metrics)
	s.db.Model(&agent).Update("metrics", metricsJSON)
}

// GetExecutionHistory returns execution history for an agent
func (s *CustomAgentService) GetExecutionHistory(ctx context.Context, agentID string, userID uint, limit int) ([]models.AgentExecution, error) {
	var executions []models.AgentExecution
	if err := s.db.WithContext(ctx).
		Where("agent_id = ? AND user_id = ?", agentID, userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}
	return executions, nil
}

// ProvideFeedback records user feedback for an execution
func (s *CustomAgentService) ProvideFeedback(ctx context.Context, executionID string, userID uint, rating int, note string) error {
	result := s.db.WithContext(ctx).
		Model(&models.AgentExecution{}).
		Where("id = ? AND user_id = ?", executionID, userID).
		Updates(map[string]interface{}{
			"feedback":      rating,
			"feedback_note": note,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to save feedback: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("execution not found")
	}
	return nil
}

// ExportAgent exports agent configuration as JSON
func (s *CustomAgentService) ExportAgent(ctx context.Context, id string, userID uint) (map[string]interface{}, error) {
	agent, err := s.GetAgent(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	export := map[string]interface{}{
		"manifest": "1.0",
		"agent": map[string]interface{}{
			"name":                  agent.Name,
			"description":           agent.Description,
			"version":               agent.Version,
			"cognitiveArchitecture": agent.CognitiveArchitecture,
			"icon":                  agent.Icon,
		},
	}

	// Add nested JSON fields
	var tags, prompt, persona, memory, tools, selfImprove, modelPrefs, inputSchema, outputSchema interface{}
	json.Unmarshal(agent.Tags, &tags)
	json.Unmarshal(agent.PromptTemplate, &prompt)
	json.Unmarshal(agent.Persona, &persona)
	json.Unmarshal(agent.MemoryConfig, &memory)
	json.Unmarshal(agent.ToolBindings, &tools)
	json.Unmarshal(agent.SelfImproveConfig, &selfImprove)
	json.Unmarshal(agent.ModelPreferences, &modelPrefs)
	json.Unmarshal(agent.InputSchema, &inputSchema)
	json.Unmarshal(agent.OutputSchema, &outputSchema)

	agentData := export["agent"].(map[string]interface{})
	agentData["tags"] = tags
	agentData["promptTemplate"] = prompt
	agentData["persona"] = persona
	agentData["memoryConfig"] = memory
	agentData["toolBindings"] = tools
	agentData["selfImproveConfig"] = selfImprove
	agentData["modelPreferences"] = modelPrefs
	agentData["inputSchema"] = inputSchema
	agentData["outputSchema"] = outputSchema

	return export, nil
}

// ImportAgent imports an agent from exported JSON
func (s *CustomAgentService) ImportAgent(ctx context.Context, userID uint, data map[string]interface{}) (*models.CustomAgent, error) {
	agentData, ok := data["agent"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid import format: missing agent data")
	}

	agent := &models.CustomAgent{
		UserID: userID,
	}

	if name, ok := agentData["name"].(string); ok {
		agent.Name = name + " (Imported)"
	}
	if desc, ok := agentData["description"].(string); ok {
		agent.Description = desc
	}
	if arch, ok := agentData["cognitiveArchitecture"].(string); ok {
		agent.CognitiveArchitecture = arch
	}
	if icon, ok := agentData["icon"].(string); ok {
		agent.Icon = icon
	}

	// Marshal nested fields back to JSON
	if tags, ok := agentData["tags"]; ok {
		tagsJSON, _ := json.Marshal(tags)
		agent.Tags = tagsJSON
	}
	if prompt, ok := agentData["promptTemplate"]; ok {
		promptJSON, _ := json.Marshal(prompt)
		agent.PromptTemplate = promptJSON
	}
	if persona, ok := agentData["persona"]; ok {
		personaJSON, _ := json.Marshal(persona)
		agent.Persona = personaJSON
	}
	if memory, ok := agentData["memoryConfig"]; ok {
		memoryJSON, _ := json.Marshal(memory)
		agent.MemoryConfig = memoryJSON
	}
	if tools, ok := agentData["toolBindings"]; ok {
		toolsJSON, _ := json.Marshal(tools)
		agent.ToolBindings = toolsJSON
	}
	if selfImprove, ok := agentData["selfImproveConfig"]; ok {
		selfImproveJSON, _ := json.Marshal(selfImprove)
		agent.SelfImproveConfig = selfImproveJSON
	}
	if modelPrefs, ok := agentData["modelPreferences"]; ok {
		modelPrefsJSON, _ := json.Marshal(modelPrefs)
		agent.ModelPreferences = modelPrefsJSON
	}
	if inputSchema, ok := agentData["inputSchema"]; ok {
		inputSchemaJSON, _ := json.Marshal(inputSchema)
		agent.InputSchema = inputSchemaJSON
	}
	if outputSchema, ok := agentData["outputSchema"]; ok {
		outputSchemaJSON, _ := json.Marshal(outputSchema)
		agent.OutputSchema = outputSchemaJSON
	}

	return s.CreateAgent(ctx, userID, agent)
}
