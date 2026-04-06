package models

import (
	"time"

	"gorm.io/gorm"
)

// ==================== Custom Agent Models ====================

// CustomAgent represents a user-defined agent with visual builder configuration
type CustomAgent struct {
	ID                    string         `gorm:"primarykey;size:36" json:"id"`
	UserID                uint           `gorm:"index" json:"user_id"`
	Name                  string         `gorm:"size:100" json:"name"`
	Description           string         `gorm:"size:500" json:"description"`
	Icon                  string         `gorm:"size:100" json:"icon"`                                  // Icon identifier or emoji
	CognitiveArchitecture string         `gorm:"size:50;default:'react'" json:"cognitive_architecture"` // react, reflexion, tot, cot, custom
	Status                string         `gorm:"size:20;default:'draft'" json:"status"`                 // draft, active, archived
	IsPublic              bool           `gorm:"default:false" json:"is_public"`                        // Shared to marketplace
	Version               string         `gorm:"size:20;default:'1.0.0'" json:"version"`                // Semantic version
	Tags                  JSON           `gorm:"type:jsonb" json:"tags"`                                // ["coding", "analysis"]
	PromptTemplate        JSON           `gorm:"type:jsonb" json:"prompt_template"`                     // {system, user, fewShotExamples[]}
	Persona               JSON           `gorm:"type:jsonb" json:"persona"`                             // {expertise[], style, tone, constraints}
	MemoryConfig          JSON           `gorm:"type:jsonb" json:"memory_config"`                       // {contextWindow, vectorStore, episodicBuffer}
	ToolBindings          JSON           `gorm:"type:jsonb" json:"tool_bindings"`                       // ToolPermission[]
	SelfImproveConfig     JSON           `gorm:"type:jsonb" json:"self_improve_config"`                 // {enabled, feedbackThreshold, maxRevisions}
	ModelPreferences      JSON           `gorm:"type:jsonb" json:"model_preferences"`                   // {preferredModel, fallbackModels[], minCapability}
	InputSchema           JSON           `gorm:"type:jsonb" json:"input_schema"`                        // JSON Schema for expected inputs
	OutputSchema          JSON           `gorm:"type:jsonb" json:"output_schema"`                       // JSON Schema for outputs
	Metrics               JSON           `gorm:"type:jsonb" json:"metrics"`                             // {successRate, avgLatency, totalExecutions}
	ParentAgentID         *string        `gorm:"size:36" json:"parent_agent_id"`                        // Forked from
	MarketplaceID         *string        `gorm:"size:36" json:"marketplace_id"`                         // Reference to marketplace listing
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

// AgentExecution records each agent run for analytics and self-improvement
type AgentExecution struct {
	ID             string    `gorm:"primarykey;size:36" json:"id"`
	AgentID        string    `gorm:"index;size:36" json:"agent_id"`
	UserID         uint      `gorm:"index" json:"user_id"`
	WorkflowID     *string   `gorm:"size:36" json:"workflow_id"` // If part of workflow
	WorkflowStepID *string   `gorm:"size:36" json:"workflow_step_id"`
	Input          string    `gorm:"type:text" json:"input"`
	Output         string    `gorm:"type:text" json:"output"`
	Model          string    `gorm:"size:50" json:"model"`
	TokensUsed     int       `json:"tokens_used"`
	LatencyMs      int       `json:"latency_ms"`
	Status         string    `gorm:"size:20" json:"status"` // success, error, timeout
	ErrorMessage   string    `gorm:"size:500" json:"error_message"`
	ToolsCalled    JSON      `gorm:"type:jsonb" json:"tools_called"` // [{tool, args, result}]
	Feedback       *int      `json:"feedback"`                       // 1-5 rating from user
	FeedbackNote   string    `gorm:"size:500" json:"feedback_note"`
	CreatedAt      time.Time `json:"created_at"`
}

// ==================== Workflow Models ====================

// Workflow represents a DAG-based multi-agent pipeline
type Workflow struct {
	ID            string         `gorm:"primarykey;size:36" json:"id"`
	UserID        uint           `gorm:"index" json:"user_id"`
	Name          string         `gorm:"size:100" json:"name"`
	Description   string         `gorm:"size:500" json:"description"`
	Icon          string         `gorm:"size:100" json:"icon"`
	Status        string         `gorm:"size:20;default:'draft'" json:"status"` // draft, active, archived
	IsPublic      bool           `gorm:"default:false" json:"is_public"`
	Version       string         `gorm:"size:20;default:'1.0.0'" json:"version"`
	Tags          JSON           `gorm:"type:jsonb" json:"tags"`
	Triggers      JSON           `gorm:"type:jsonb" json:"triggers"`      // ["manual", "schedule", "webhook", "event"]
	GlobalConfig  JSON           `gorm:"type:jsonb" json:"global_config"` // {timeout, retryPolicy, notifyOnComplete}
	InputSchema   JSON           `gorm:"type:jsonb" json:"input_schema"`  // Expected workflow inputs
	OutputSchema  JSON           `gorm:"type:jsonb" json:"output_schema"` // Expected workflow outputs
	Variables     JSON           `gorm:"type:jsonb" json:"variables"`     // Workflow-level variables
	Metrics       JSON           `gorm:"type:jsonb" json:"metrics"`       // {successRate, avgDuration, totalRuns}
	ParentID      *string        `gorm:"size:36" json:"parent_id"`        // Forked from
	MarketplaceID *string        `gorm:"size:36" json:"marketplace_id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Steps         []WorkflowStep `gorm:"foreignKey:WorkflowID" json:"steps,omitempty"`
	Edges         []WorkflowEdge `gorm:"foreignKey:WorkflowID" json:"edges,omitempty"`
}

// WorkflowStep represents a single node in the workflow DAG
type WorkflowStep struct {
	ID             string         `gorm:"primarykey;size:36" json:"id"`
	WorkflowID     string         `gorm:"index;size:36" json:"workflow_id"`
	Name           string         `gorm:"size:100" json:"name"`
	Type           string         `gorm:"size:30" json:"type"`              // agent, tool, condition, checkpoint, subworkflow, transform
	AgentID        *string        `gorm:"size:36" json:"agent_id"`          // Reference to CustomAgent
	ToolID         *string        `gorm:"size:100" json:"tool_id"`          // MCP tool identifier
	SubWorkflowID  *string        `gorm:"size:36" json:"sub_workflow_id"`   // For recursive workflows
	Config         JSON           `gorm:"type:jsonb" json:"config"`         // Step-specific configuration
	InputMapping   JSON           `gorm:"type:jsonb" json:"input_mapping"`  // Map workflow vars to step inputs
	OutputMapping  JSON           `gorm:"type:jsonb" json:"output_mapping"` // Map step outputs to workflow vars
	Condition      JSON           `gorm:"type:jsonb" json:"condition"`      // For conditional branching
	RetryConfig    JSON           `gorm:"type:jsonb" json:"retry_config"`   // {maxRetries, backoffMs}
	TimeoutSeconds int            `gorm:"default:300" json:"timeout_seconds"`
	IsParallel     bool           `gorm:"default:false" json:"is_parallel"` // Can run in parallel lane
	ParallelGroup  string         `gorm:"size:50" json:"parallel_group"`    // Group ID for parallel steps
	PositionX      int            `json:"position_x"`                       // Visual position
	PositionY      int            `json:"position_y"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// WorkflowEdge represents a connection between steps in the DAG
type WorkflowEdge struct {
	ID           string         `gorm:"primarykey;size:36" json:"id"`
	WorkflowID   string         `gorm:"index;size:36" json:"workflow_id"`
	SourceStepID string         `gorm:"size:36" json:"source_step_id"`
	TargetStepID string         `gorm:"size:36" json:"target_step_id"`
	Label        string         `gorm:"size:50" json:"label"`        // "success", "error", "true", "false"
	Condition    JSON           `gorm:"type:jsonb" json:"condition"` // Condition to traverse this edge
	Priority     int            `gorm:"default:0" json:"priority"`   // For ordering when multiple edges
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// WorkflowRun tracks a single execution of a workflow
type WorkflowRun struct {
	ID             string     `gorm:"primarykey;size:36" json:"id"`
	WorkflowID     string     `gorm:"index;size:36" json:"workflow_id"`
	UserID         uint       `gorm:"index" json:"user_id"`
	Status         string     `gorm:"size:20" json:"status"`       // pending, running, paused, completed, failed, cancelled
	TriggerType    string     `gorm:"size:20" json:"trigger_type"` // manual, schedule, webhook, event
	Input          JSON       `gorm:"type:jsonb" json:"input"`
	Output         JSON       `gorm:"type:jsonb" json:"output"`
	Variables      JSON       `gorm:"type:jsonb" json:"variables"` // Runtime variables
	CurrentStepID  *string    `gorm:"size:36" json:"current_step_id"`
	CheckpointData JSON       `gorm:"type:jsonb" json:"checkpoint_data"` // For human-in-loop
	ErrorMessage   string     `gorm:"size:500" json:"error_message"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	PausedAt       *time.Time `json:"paused_at"`
	DurationMs     int        `json:"duration_ms"`
	CreatedAt      time.Time  `json:"created_at"`
}

// WorkflowStepRun tracks execution of a single step within a run
type WorkflowStepRun struct {
	ID            string     `gorm:"primarykey;size:36" json:"id"`
	WorkflowRunID string     `gorm:"index;size:36" json:"workflow_run_id"`
	StepID        string     `gorm:"index;size:36" json:"step_id"`
	Status        string     `gorm:"size:20" json:"status"` // pending, running, completed, failed, skipped
	Input         JSON       `gorm:"type:jsonb" json:"input"`
	Output        JSON       `gorm:"type:jsonb" json:"output"`
	ErrorMessage  string     `gorm:"size:500" json:"error_message"`
	RetryCount    int        `gorm:"default:0" json:"retry_count"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	DurationMs    int        `json:"duration_ms"`
}

// ==================== Permission Models ====================

// ToolPermission defines granular access control for tools
type ToolPermission struct {
	ID               string         `gorm:"primarykey;size:36" json:"id"`
	UserID           uint           `gorm:"index" json:"user_id"`
	Name             string         `gorm:"size:100" json:"name"`
	Description      string         `gorm:"size:500" json:"description"`
	ToolPattern      string         `gorm:"size:200" json:"tool_pattern"`   // "file/*", "mcp://github/*"
	Actions          JSON           `gorm:"type:jsonb" json:"actions"`      // ["read", "write", "execute"]
	Scope            string         `gorm:"size:20" json:"scope"`           // global, agent, workflow, user
	ScopeID          *string        `gorm:"size:36" json:"scope_id"`        // Reference to agent/workflow if scoped
	RateLimit        JSON           `gorm:"type:jsonb" json:"rate_limit"`   // {maxPerMinute, maxPerHour, maxPerDay}
	CostLimit        JSON           `gorm:"type:jsonb" json:"cost_limit"`   // {maxCostPerRequest, maxCostPerDay}
	AllowedArgs      JSON           `gorm:"type:jsonb" json:"allowed_args"` // Regex patterns for allowed args
	BlockedArgs      JSON           `gorm:"type:jsonb" json:"blocked_args"` // Regex patterns to block
	RequiresApproval bool           `gorm:"default:false" json:"requires_approval"`
	AuditLevel       string         `gorm:"size:20;default:'summary'" json:"audit_level"` // none, summary, detailed
	IsEnabled        bool           `gorm:"default:true" json:"is_enabled"`
	ExpiresAt        *time.Time     `json:"expires_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// ToolInvocationLog records every tool call for auditing
type ToolInvocationLog struct {
	ID           string    `gorm:"primarykey;size:36" json:"id"`
	UserID       uint      `gorm:"index" json:"user_id"`
	AgentID      *string   `gorm:"size:36" json:"agent_id"`
	WorkflowID   *string   `gorm:"size:36" json:"workflow_id"`
	ToolName     string    `gorm:"size:200" json:"tool_name"`
	ToolArgs     JSON      `gorm:"type:jsonb" json:"tool_args"`
	Result       string    `gorm:"type:text" json:"result"`
	Status       string    `gorm:"size:20" json:"status"`        // allowed, denied, rate_limited, error
	PermissionID *string   `gorm:"size:36" json:"permission_id"` // Which permission was checked
	DenialReason string    `gorm:"size:200" json:"denial_reason"`
	CostEstimate float64   `gorm:"default:0" json:"cost_estimate"`
	LatencyMs    int       `json:"latency_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

// ==================== Marketplace Models ====================

// MarketplaceItem represents a published agent or workflow
type MarketplaceItem struct {
	ID             string         `gorm:"primarykey;size:36" json:"id"`
	AuthorID       uint           `gorm:"index" json:"author_id"`
	AuthorName     string         `gorm:"size:100" json:"author_name"`
	Type           string         `gorm:"size:20" json:"type"`      // agent, workflow
	SourceID       string         `gorm:"size:36" json:"source_id"` // CustomAgent.ID or Workflow.ID
	Name           string         `gorm:"size:100" json:"name"`
	Description    string         `gorm:"type:text" json:"description"`
	Icon           string         `gorm:"size:100" json:"icon"`
	Version        string         `gorm:"size:20" json:"version"`
	License        string         `gorm:"size:50;default:'MIT'" json:"license"`
	Tags           JSON           `gorm:"type:jsonb" json:"tags"`
	Category       string         `gorm:"size:50" json:"category"`          // coding, writing, analysis, etc.
	RequiredTools  JSON           `gorm:"type:jsonb" json:"required_tools"` // Tools needed to run
	MinModelCap    string         `gorm:"size:50" json:"min_model_capability"`
	Documentation  string         `gorm:"type:text" json:"documentation"` // Full docs in markdown
	Screenshots    JSON           `gorm:"type:jsonb" json:"screenshots"`  // Image URLs
	DemoVideo      string         `gorm:"size:500" json:"demo_video"`
	PackageData    JSON           `gorm:"type:jsonb" json:"package_data"`          // Full exportable config
	Status         string         `gorm:"size:20;default:'pending'" json:"status"` // pending, published, rejected, archived
	ReviewNotes    string         `gorm:"size:500" json:"review_notes"`
	Downloads      int            `gorm:"default:0" json:"downloads"`
	Stars          int            `gorm:"default:0" json:"stars"`
	ForkCount      int            `gorm:"default:0" json:"fork_count"`
	AvgRating      float64        `gorm:"default:0" json:"avg_rating"`
	RatingCount    int            `gorm:"default:0" json:"rating_count"`
	SuccessRate    float64        `gorm:"default:0" json:"success_rate"`
	AvgLatencyMs   int            `gorm:"default:0" json:"avg_latency_ms"`
	BenchmarkScore float64        `gorm:"default:0" json:"benchmark_score"` // 0-100 quality score
	IsFeatured     bool           `gorm:"default:false" json:"is_featured"`
	PublishedAt    *time.Time     `json:"published_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// MarketplaceReview stores user reviews
type MarketplaceReview struct {
	ID           string    `gorm:"primarykey;size:36" json:"id"`
	ItemID       string    `gorm:"index;size:36" json:"item_id"`
	UserID       uint      `gorm:"index" json:"user_id"`
	Username     string    `gorm:"size:50" json:"username"`
	Rating       int       `json:"rating"` // 1-5
	Title        string    `gorm:"size:200" json:"title"`
	Comment      string    `gorm:"type:text" json:"comment"`
	IsVerified   bool      `gorm:"default:false" json:"is_verified"` // User actually used it
	HelpfulCount int       `gorm:"default:0" json:"helpful_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// MarketplaceStar tracks who starred what
type MarketplaceStar struct {
	ID        string    `gorm:"primarykey;size:36" json:"id"`
	ItemID    string    `gorm:"uniqueIndex:idx_star_user_item;size:36" json:"item_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_star_user_item" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// MarketplaceDownload tracks downloads
type MarketplaceDownload struct {
	ID        string    `gorm:"primarykey;size:36" json:"id"`
	ItemID    string    `gorm:"index;size:36" json:"item_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Version   string    `gorm:"size:20" json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// ==================== A/B Testing Models ====================

// ABTest represents an A/B test between agents or workflows
type ABTest struct {
	ID            string         `gorm:"primarykey;size:36" json:"id"`
	UserID        uint           `gorm:"index" json:"user_id"`
	Name          string         `gorm:"size:100" json:"name"`
	Description   string         `gorm:"size:500" json:"description"`
	Type          string         `gorm:"size:20" json:"type"` // agent, workflow
	VariantAID    string         `gorm:"size:36" json:"variant_a_id"`
	VariantBID    string         `gorm:"size:36" json:"variant_b_id"`
	TrafficSplit  int            `gorm:"default:50" json:"traffic_split"`       // % to variant A
	Status        string         `gorm:"size:20;default:'draft'" json:"status"` // draft, running, completed, cancelled
	WinnerID      *string        `gorm:"size:36" json:"winner_id"`
	WinCriteria   JSON           `gorm:"type:jsonb" json:"win_criteria"` // {metric: "successRate", threshold: 0.05}
	MinSampleSize int            `gorm:"default:100" json:"min_sample_size"`
	ResultsA      JSON           `gorm:"type:jsonb" json:"results_a"` // {runs, successes, avgLatency, avgRating}
	ResultsB      JSON           `gorm:"type:jsonb" json:"results_b"`
	StartedAt     *time.Time     `json:"started_at"`
	CompletedAt   *time.Time     `json:"completed_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// ABTestRun tracks individual runs in an A/B test
type ABTestRun struct {
	ID          string    `gorm:"primarykey;size:36" json:"id"`
	TestID      string    `gorm:"index;size:36" json:"test_id"`
	VariantID   string    `gorm:"size:36" json:"variant_id"`   // A or B
	ExecutionID string    `gorm:"size:36" json:"execution_id"` // AgentExecution or WorkflowRun ID
	Success     bool      `json:"success"`
	LatencyMs   int       `json:"latency_ms"`
	UserRating  *int      `json:"user_rating"`
	CreatedAt   time.Time `json:"created_at"`
}
