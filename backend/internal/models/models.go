package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type JSON = datatypes.JSON

// User model
type User struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	UUID       string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Username   string         `gorm:"uniqueIndex;size:50" json:"username"`
	Email      string         `gorm:"uniqueIndex;size:100" json:"email"`
	Phone      string         `gorm:"size:20" json:"phone"`
	Password   string         `gorm:"size:255" json:"-"`
	Avatar     string         `gorm:"size:500" json:"avatar"`
	Settings   JSON           `gorm:"type:jsonb" json:"settings"` // User preferences
	Role       string         `gorm:"size:20;default:'user'" json:"role"`
	LastActive time.Time      `json:"last_active"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// Conversation model
type Conversation struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	UserID         uint           `gorm:"index" json:"user_id"`
	Title          string         `gorm:"size:200" json:"title"`      // AI generated title
	ModelType      string         `gorm:"size:50" json:"model_type"`  // AI model used
	Settings       JSON           `gorm:"type:jsonb" json:"settings"` // Conversation settings
	MessageCount   int            `gorm:"default:0" json:"message_count"`
	LastMessage    string         `gorm:"size:500" json:"last_message"`
	IsPinned       bool           `gorm:"default:false" json:"is_pinned"`
	IsArchived     bool           `gorm:"default:false" json:"is_archived"`
	ActiveBranchID *string        `gorm:"size:36" json:"active_branch_id"` // Currently active branch
	BranchCount    int            `gorm:"default:0" json:"branch_count"`   // Number of branches
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Messages       []Message      `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

// Message model
type Message struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	ConversationID uint           `gorm:"index" json:"conversation_id"`
	Role           string         `gorm:"size:20" json:"role"` // user/assistant/system
	Content        string         `gorm:"type:text" json:"content"`
	Tokens         int            `gorm:"default:0" json:"tokens"`
	Model          string         `gorm:"size:50" json:"model"`
	Status         string         `gorm:"size:20" json:"status"` // pending/success/error
	Error          string         `gorm:"size:500" json:"error"`
	ParentID       *uint          `json:"parent_id"`                      // For message tree
	BranchID       *string        `gorm:"size:36;index" json:"branch_id"` // Branch this message belongs to
	Version        int            `gorm:"default:1" json:"version"`       // Edit version number
	OriginalID     *uint          `json:"original_id"`                    // Points to original message if edited
	IsEdited       bool           `gorm:"default:false" json:"is_edited"` // Has been edited
	Metadata       JSON           `gorm:"type:jsonb" json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	IndexedAt      *time.Time     `gorm:"index" json:"indexed_at"` // Vector index time
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// Document model (for RAG Knowledge Base)
type Document struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index" json:"user_id"`
	Title      string    `gorm:"size:200" json:"title"`
	Filename   string    `gorm:"size:255" json:"filename"`
	FileType   string    `gorm:"size:20" json:"file_type"` // pdf, txt, md, docx
	FileSize   int64     `json:"file_size"`
	Content    string    `gorm:"type:text" json:"content"`
	Status     string    `gorm:"size:20;default:'processing'" json:"status"` // processing, ready, failed
	ChunkCount int       `json:"chunk_count"`
	ErrorMsg   string    `gorm:"size:500" json:"error_msg,omitempty"`
	Metadata   JSON      `gorm:"type:jsonb" json:"metadata"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ProviderSetting model
type ProviderSetting struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	UserID    uint           `gorm:"uniqueIndex:idx_user_provider" json:"user_id"`
	Provider  string         `gorm:"uniqueIndex:idx_user_provider;size:50" json:"provider"`          // aliyun, deepseek, tencent, etc.
	Type      string         `gorm:"uniqueIndex:idx_user_provider;size:20;default:'ai'" json:"type"` // ai, search
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	APIKey    string         `gorm:"size:255" json:"api_key"`
	SecretKey string         `gorm:"size:255" json:"secret_key,omitempty"`
	SecretID  string         `gorm:"size:255" json:"secret_id,omitempty"`
	BaseURL   string         `gorm:"size:255" json:"base_url"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
