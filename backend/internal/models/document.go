package models

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Vector is a custom type for pgvector that properly serializes []float32
type Vector []float32

// Value implements driver.Valuer for Vector - serializes to PostgreSQL vector string format
func (v Vector) Value() (driver.Value, error) {
	if v == nil {
		return nil, nil
	}
	var builder strings.Builder
	builder.WriteString("[")
	for i, val := range v {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(strconv.FormatFloat(float64(val), 'f', -1, 32))
	}
	builder.WriteString("]")
	return builder.String(), nil
}

// Scan implements sql.Scanner for Vector - deserializes from PostgreSQL vector format
func (v *Vector) Scan(src interface{}) error {
	if src == nil {
		*v = nil
		return nil
	}

	var str string
	switch val := src.(type) {
	case []byte:
		str = string(val)
	case string:
		str = val
	default:
		return fmt.Errorf("unsupported type for Vector: %T", src)
	}

	// Parse [x,y,z,...] format
	str = strings.TrimPrefix(str, "[")
	str = strings.TrimSuffix(str, "]")
	if str == "" {
		*v = nil
		return nil
	}

	parts := strings.Split(str, ",")
	result := make(Vector, len(parts))
	for i, part := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return fmt.Errorf("failed to parse vector element: %w", err)
		}
		result[i] = float32(f)
	}
	*v = result
	return nil
}

// DocumentChunk represents a chunk of a document with its embedding
type DocumentChunk struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	DocumentID string    `json:"document_id" gorm:"index"`
	Content    string    `json:"content"`
	ChunkIndex int       `json:"chunk_index"`
	TokenCount int       `json:"token_count"`
	Embedding  Vector    `json:"-" gorm:"type:vector(1024)"`
	CreatedAt  time.Time `json:"created_at"`
}

// RAGQuery represents a query to the knowledge base
type RAGQuery struct {
	Query     string   `json:"query" binding:"required"`
	TopK      int      `json:"top_k"`     // Number of results to return
	Threshold float64  `json:"threshold"` // Similarity threshold (0-1)
	DocIDs    []string `json:"doc_ids"`   // Optional: filter by document IDs
}

// RAGResult represents a search result from the knowledge base
type RAGResult struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Filename   string  `json:"filename"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	ChunkIndex int     `json:"chunk_index"`
}

// RAGContext represents context to be injected into a chat prompt
type RAGContext struct {
	Query   string      `json:"query"`
	Results []RAGResult `json:"results"`
}

// Document status constants
const (
	DocStatusProcessing = "processing"
	DocStatusReady      = "ready"
	DocStatusFailed     = "failed"
)
