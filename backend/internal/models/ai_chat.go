package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionStatusPending    SessionStatus = "pending"
	SessionStatusRunning    SessionStatus = "running"
	SessionStatusPaused     SessionStatus = "paused"
	SessionStatusCompleted  SessionStatus = "completed"
	SessionStatusError      SessionStatus = "error"
	SessionStatusTerminated SessionStatus = "terminated"
)

// TerminationType 终止类型
type TerminationType string

const (
	TerminationTypeFixedRounds TerminationType = "fixed_rounds"
	TerminationTypeOpenEnded   TerminationType = "open_ended"
	TerminationTypeKeyword     TerminationType = "keyword"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeToolCall    MessageType = "tool_call"
	MessageTypeToolResult  MessageType = "tool_result"
	MessageTypeDirectorCmd MessageType = "director_cmd"
	MessageTypeSystem      MessageType = "system"
)

// LanguageStyle 语言风格
type LanguageStyle string

const (
	LanguageStyleProfessional LanguageStyle = "professional"
	LanguageStyleCasual       LanguageStyle = "casual"
	LanguageStylePoetic       LanguageStyle = "poetic"
	LanguageStyleAcademic     LanguageStyle = "academic"
	LanguageStyleHumorous     LanguageStyle = "humorous"
)

// KnowledgeLevel 知识水平
type KnowledgeLevel string

const (
	KnowledgeLevelBeginner     KnowledgeLevel = "beginner"
	KnowledgeLevelIntermediate KnowledgeLevel = "intermediate"
	KnowledgeLevelExpert       KnowledgeLevel = "expert"
)

// StringArray JSON 数组类型
type StringArray []string

// Value 实现 driver.Valuer 接口
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// Scan 实现 sql.Scanner 接口
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("cannot scan type into StringArray")
	}

	return json.Unmarshal(bytes, a)
}

// JSONMap JSON 对象类型
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 实现 sql.Scanner 接口
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("cannot scan type into JSONMap")
	}

	return json.Unmarshal(bytes, m)
}

// StyleConfig 风格配置
type StyleConfig struct {
	LanguageStyle  LanguageStyle  `json:"language_style" gorm:"column:style_language"`
	KnowledgeLevel KnowledgeLevel `json:"knowledge_level" gorm:"column:style_knowledge"`
	Tone           string         `json:"tone" gorm:"column:style_tone"`
}

// TerminationConfig 终止条件配置
type TerminationConfig struct {
	Type                     TerminationType `json:"type" gorm:"column:termination_type"`
	MaxRounds                int             `json:"max_rounds" gorm:"column:termination_max_rounds"`
	Keywords                 StringArray     `json:"keywords" gorm:"column:termination_keywords;type:json"`
	SimilarityThreshold      float64         `json:"similarity_threshold" gorm:"column:termination_similarity_threshold"`
	ConsecutiveSimilarRounds int             `json:"consecutive_similar_rounds" gorm:"column:termination_consecutive_similar_rounds"`
}

// AgentConfig AI 代理配置
type AgentConfig struct {
	Name         string      `json:"name"`
	Role         string      `json:"role"`
	Style        StyleConfig `json:"style" gorm:"embedded"`
	Model        string      `json:"model"`
	Temperature  float64     `json:"temperature"`
	MaxTokens    int         `json:"max_tokens"`
	AllowedTools StringArray `json:"allowed_tools" gorm:"type:json"`
	BlockedTools StringArray `json:"blocked_tools" gorm:"type:json"`
}

// TokenUsage Token 使用统计
type TokenUsage struct {
	AgentAInput  int `json:"agent_a_input" gorm:"column:token_agent_a_input"`
	AgentAOutput int `json:"agent_a_output" gorm:"column:token_agent_a_output"`
	AgentBInput  int `json:"agent_b_input" gorm:"column:token_agent_b_input"`
	AgentBOutput int `json:"agent_b_output" gorm:"column:token_agent_b_output"`
	Total        int `json:"total" gorm:"column:token_total"`
}

// AgentTokenUsage tracks token usage per agent in multi-agent sessions
type AgentTokenUsage struct {
	AgentID      string `json:"agent_id"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// AgentConfigArray is a JSON array of agent configs for multi-agent support
type AgentConfigArray []AgentConfig

// Value implements driver.Valuer for JSON serialization
func (a AgentConfigArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// Scan implements sql.Scanner for JSON deserialization
func (a *AgentConfigArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("cannot scan type into AgentConfigArray")
	}

	return json.Unmarshal(bytes, a)
}

// AIChatSession AI-AI 聊天会话
type AIChatSession struct {
	ID                string            `json:"id" gorm:"primaryKey;size:255"`
	Title             string            `json:"title" gorm:"size:500"`
	Status            SessionStatus     `json:"status" gorm:"size:50;default:'pending'"`
	CurrentRound      int               `json:"current_round" gorm:"default:0"`
	MaxRounds         int               `json:"max_rounds" gorm:"default:10"`
	Topic             string            `json:"topic" gorm:"type:text"`
	GlobalConstraint  string            `json:"global_constraint" gorm:"type:text"`
	TerminationConfig TerminationConfig `json:"termination_config" gorm:"embedded"`

	// Multi-agent support (3+ agents)
	Agents         AgentConfigArray `json:"agents" gorm:"type:json"`          // Array of agent configs for multi-agent
	SpeakingOrder  StringArray      `json:"speaking_order" gorm:"type:json"`  // Custom speaking order (agent IDs)
	AgentCount     int              `json:"agent_count" gorm:"default:2"`     // Number of agents
	CurrentSpeaker int              `json:"current_speaker" gorm:"default:0"` // Index of current speaker

	// Agent A 配置 - 使用嵌入前缀 (backward compatibility)
	AgentAName         string      `json:"agent_a_name" gorm:"column:agent_a_name;size:255"`
	AgentARole         string      `json:"agent_a_role" gorm:"column:agent_a_role;type:text"`
	AgentAStyle        StyleConfig `json:"agent_a_style" gorm:"embedded;embeddedPrefix:agent_a_"`
	AgentAModel        string      `json:"agent_a_model" gorm:"column:agent_a_model;size:100"`
	AgentATemperature  float64     `json:"agent_a_temperature" gorm:"column:agent_a_temperature"`
	AgentAMaxTokens    int         `json:"agent_a_max_tokens" gorm:"column:agent_a_max_tokens"`
	AgentAAllowedTools StringArray `json:"agent_a_allowed_tools" gorm:"column:agent_a_allowed_tools;type:json"`
	AgentABlockedTools StringArray `json:"agent_a_blocked_tools" gorm:"column:agent_a_blocked_tools;type:json"`

	// Agent B 配置 - 使用嵌入前缀
	AgentBName         string      `json:"agent_b_name" gorm:"column:agent_b_name;size:255"`
	AgentBRole         string      `json:"agent_b_role" gorm:"column:agent_b_role;type:text"`
	AgentBStyle        StyleConfig `json:"agent_b_style" gorm:"embedded;embeddedPrefix:agent_b_"`
	AgentBModel        string      `json:"agent_b_model" gorm:"column:agent_b_model;size:100"`
	AgentBTemperature  float64     `json:"agent_b_temperature" gorm:"column:agent_b_temperature"`
	AgentBMaxTokens    int         `json:"agent_b_max_tokens" gorm:"column:agent_b_max_tokens"`
	AgentBAllowedTools StringArray `json:"agent_b_allowed_tools" gorm:"column:agent_b_allowed_tools;type:json"`
	AgentBBlockedTools StringArray `json:"agent_b_blocked_tools" gorm:"column:agent_b_blocked_tools;type:json"`

	// 分支管理
	ParentID    *string `json:"parent_id,omitempty" gorm:"size:255"`
	BranchPoint *int    `json:"branch_point,omitempty"`

	// Token 统计
	TokenUsage TokenUsage `json:"token_usage" gorm:"embedded"`

	// 时间戳
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	// 关联
	Messages []AIChatMessage `json:"messages,omitempty" gorm:"foreignKey:SessionID;references:ID"`
}

// TableName 指定表名
func (AIChatSession) TableName() string {
	return "ai_chat_sessions"
}

// GetAgentConfig 获取 Agent 配置
func (s *AIChatSession) GetAgentConfig(agentID string) AgentConfig {
	if agentID == "agent_a" {
		return AgentConfig{
			Name:         s.AgentAName,
			Role:         s.AgentARole,
			Style:        s.AgentAStyle,
			Model:        s.AgentAModel,
			Temperature:  s.AgentATemperature,
			MaxTokens:    s.AgentAMaxTokens,
			AllowedTools: s.AgentAAllowedTools,
			BlockedTools: s.AgentABlockedTools,
		}
	}
	return AgentConfig{
		Name:         s.AgentBName,
		Role:         s.AgentBRole,
		Style:        s.AgentBStyle,
		Model:        s.AgentBModel,
		Temperature:  s.AgentBTemperature,
		MaxTokens:    s.AgentBMaxTokens,
		AllowedTools: s.AgentBAllowedTools,
		BlockedTools: s.AgentBBlockedTools,
	}
}

// SetAgentConfig 设置 Agent 配置
func (s *AIChatSession) SetAgentConfig(agentID string, config AgentConfig) {
	if agentID == "agent_a" {
		s.AgentAName = config.Name
		s.AgentARole = config.Role
		s.AgentAStyle = config.Style
		s.AgentAModel = config.Model
		s.AgentATemperature = config.Temperature
		s.AgentAMaxTokens = config.MaxTokens
		s.AgentAAllowedTools = config.AllowedTools
		s.AgentABlockedTools = config.BlockedTools
	} else {
		s.AgentBName = config.Name
		s.AgentBRole = config.Role
		s.AgentBStyle = config.Style
		s.AgentBModel = config.Model
		s.AgentBTemperature = config.Temperature
		s.AgentBMaxTokens = config.MaxTokens
		s.AgentBAllowedTools = config.AllowedTools
		s.AgentBBlockedTools = config.BlockedTools
	}
}

// IsMultiAgent returns true if this session has more than 2 agents
func (s *AIChatSession) IsMultiAgent() bool {
	return len(s.Agents) > 2
}

// GetAgentCount returns the number of agents in the session
func (s *AIChatSession) GetAgentCount() int {
	if len(s.Agents) > 0 {
		return len(s.Agents)
	}
	return 2 // Default 2 agents (agent_a and agent_b)
}

// GetAllAgents returns all agent configs (multi-agent mode or legacy mode)
func (s *AIChatSession) GetAllAgents() []AgentConfig {
	if len(s.Agents) > 0 {
		return s.Agents
	}
	// Legacy mode: return agent_a and agent_b
	return []AgentConfig{
		s.GetAgentConfig("agent_a"),
		s.GetAgentConfig("agent_b"),
	}
}

// GetAgentByIndex returns agent config by index (for multi-agent support)
func (s *AIChatSession) GetAgentByIndex(index int) *AgentConfig {
	if len(s.Agents) > 0 && index < len(s.Agents) {
		return &s.Agents[index]
	}
	// Legacy mode
	if index == 0 {
		cfg := s.GetAgentConfig("agent_a")
		return &cfg
	}
	if index == 1 {
		cfg := s.GetAgentConfig("agent_b")
		return &cfg
	}
	return nil
}

// GetAgentID returns agent ID by index
func (s *AIChatSession) GetAgentID(index int) string {
	if len(s.Agents) > 0 && index < len(s.Agents) {
		return fmt.Sprintf("agent_%d", index)
	}
	if index == 0 {
		return "agent_a"
	}
	return "agent_b"
}

// GetNextSpeakerIndex returns the next speaker index based on speaking order
func (s *AIChatSession) GetNextSpeakerIndex(messageCount int) int {
	agentCount := s.GetAgentCount()
	if agentCount == 0 {
		return 0
	}
	return messageCount % agentCount
}

// ToolCall 工具调用
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult 工具调用结果
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
}

// AIChatMessage AI-AI 对话消息
type AIChatMessage struct {
	ID          uint        `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID   string      `json:"session_id" gorm:"size:255;index;not null"`
	Round       int         `json:"round" gorm:"not null"`
	AgentID     string      `json:"agent_id" gorm:"size:50;not null"`
	AgentName   string      `json:"agent_name" gorm:"size:255"`
	Content     string      `json:"content" gorm:"type:text"`
	MessageType MessageType `json:"message_type" gorm:"size:50;default:'text'"`
	ToolCalls   JSONMap     `json:"tool_calls" gorm:"type:json"`
	ToolResults JSONMap     `json:"tool_results" gorm:"type:json"`
	Tokens      int         `json:"tokens" gorm:"default:0"`
	Latency     int64       `json:"latency_ms" gorm:"column:latency_ms"`
	Timestamp   time.Time   `json:"timestamp" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (AIChatMessage) TableName() string {
	return "ai_chat_messages"
}

// DirectorCommand 导演指令
type DirectorCommand struct {
	ID               string    `json:"id" gorm:"primaryKey;size:255"`
	SessionID        string    `json:"session_id" gorm:"size:255;index;not null"`
	TargetAgent      string    `json:"target_agent" gorm:"size:50;not null"`
	Command          string    `json:"command" gorm:"type:text;not null"`
	InsertAfterRound *int      `json:"insert_after_round"`
	Executed         bool      `json:"executed" gorm:"default:false"`
	Timestamp        time.Time `json:"timestamp" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (DirectorCommand) TableName() string {
	return "ai_chat_director_commands"
}

// SessionSnapshot 会话快照
type SessionSnapshot struct {
	ID           string    `json:"id" gorm:"primaryKey;size:255"`
	SessionID    string    `json:"session_id" gorm:"size:255;index;not null"`
	Title        string    `json:"title" gorm:"size:500"`
	Round        int       `json:"round" gorm:"not null"`
	SnapshotData JSONMap   `json:"snapshot_data" gorm:"type:json;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (SessionSnapshot) TableName() string {
	return "ai_chat_snapshots"
}

// EvaluationReport 评估报告
type EvaluationReport struct {
	ID        string    `json:"id" gorm:"primaryKey;size:255"`
	SessionID string    `json:"session_id" gorm:"size:255;index;not null"`
	Report    JSONMap   `json:"report" gorm:"type:json;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (EvaluationReport) TableName() string {
	return "ai_chat_evaluations"
}

// AuditLog 审计日志
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID string    `json:"session_id" gorm:"size:255;index;not null"`
	EventType string    `json:"event_type" gorm:"size:100;not null"`
	EventData JSONMap   `json:"event_data" gorm:"type:json"`
	Timestamp time.Time `json:"timestamp" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "ai_chat_audit_logs"
}

// SessionTemplate 会话模板
type SessionTemplate struct {
	ID          string    `json:"id" gorm:"primaryKey;size:255"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description string    `json:"description" gorm:"type:text"`
	Icon        string    `json:"icon" gorm:"size:100"`
	Category    string    `json:"category" gorm:"size:100"`
	Config      JSONMap   `json:"config" gorm:"type:json;not null"`
	IsBuiltin   bool      `json:"is_builtin" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (SessionTemplate) TableName() string {
	return "ai_chat_templates"
}
