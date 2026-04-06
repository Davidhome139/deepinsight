package models

import (
	"time"

	"gorm.io/gorm"
)

// Branch represents a conversation branch/fork
type Branch struct {
	ID             string         `gorm:"primarykey;size:36" json:"id"`
	ConversationID uint           `gorm:"index" json:"conversation_id"`
	Name           string         `gorm:"size:100" json:"name"`
	Description    string         `gorm:"size:500" json:"description"`
	ParentBranchID *string        `gorm:"size:36" json:"parent_branch_id"`        // Parent branch (nil for main branch)
	ForkPointMsgID *uint          `json:"fork_point_message_id"`                  // Message ID where fork occurred
	IsMain         bool           `gorm:"default:false" json:"is_main"`           // Is this the main/default branch
	IsActive       bool           `gorm:"default:true" json:"is_active"`          // Is this branch currently active
	Status         string         `gorm:"size:20;default:'active'" json:"status"` // active, archived, merged
	Score          *float64       `json:"score"`                                  // AI quality score
	MessageCount   int            `gorm:"default:0" json:"message_count"`
	Metadata       JSON           `gorm:"type:jsonb" json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// MessageVersion stores edit history for messages
type MessageVersion struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	MessageID uint      `gorm:"index" json:"message_id"`
	Version   int       `gorm:"default:1" json:"version"`
	Content   string    `gorm:"type:text" json:"content"`
	EditedBy  uint      `json:"edited_by"` // User ID who made the edit
	CreatedAt time.Time `json:"created_at"`
}

// ParallelExploration represents a multi-model parallel exploration session
type ParallelExploration struct {
	ID             string         `gorm:"primarykey;size:36" json:"id"`
	ConversationID uint           `gorm:"index" json:"conversation_id"`
	SourceMsgID    uint           `json:"source_message_id"` // The user message that triggered exploration
	Prompt         string         `gorm:"type:text" json:"prompt"`
	Models         JSON           `gorm:"type:jsonb" json:"models"`                // List of models to explore
	Status         string         `gorm:"size:20;default:'pending'" json:"status"` // pending, running, completed, failed
	Results        JSON           `gorm:"type:jsonb" json:"results"`               // Array of {model, branch_id, response_preview, score}
	BestBranchID   *string        `gorm:"size:36" json:"best_branch_id"`           // Auto-selected best branch
	CreatedAt      time.Time      `json:"created_at"`
	CompletedAt    *time.Time     `json:"completed_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// ParallelResult represents a single model's result in parallel exploration
type ParallelResult struct {
	Model           string   `json:"model"`
	BranchID        string   `json:"branch_id"`
	ResponsePreview string   `json:"response_preview"` // First 200 chars
	FullResponse    string   `json:"full_response,omitempty"`
	TokenCount      int      `json:"token_count"`
	Latency         int64    `json:"latency_ms"`
	Score           *float64 `json:"score"` // AI-evaluated quality score
	Error           string   `json:"error,omitempty"`
}

// BranchTree represents the tree structure for visualization
type BranchTree struct {
	Branch   *Branch       `json:"branch"`
	Children []*BranchTree `json:"children,omitempty"`
	Messages []Message     `json:"messages,omitempty"`
}

// MessageNode represents a message in the tree view
type MessageNode struct {
	Message  *Message         `json:"message"`
	Children []*MessageNode   `json:"children,omitempty"`
	Versions []MessageVersion `json:"versions,omitempty"`
}

// BranchStatus constants
const (
	BranchStatusActive   = "active"
	BranchStatusArchived = "archived"
	BranchStatusMerged   = "merged"
)

// ParallelExplorationStatus constants
const (
	ParallelStatusPending   = "pending"
	ParallelStatusRunning   = "running"
	ParallelStatusCompleted = "completed"
	ParallelStatusFailed    = "failed"
)
