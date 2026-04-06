package common

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Messages  []Message `json:"messages"`
	Model     string    `json:"model"`
	MaxTokens int       `json:"maxTokens,omitempty"`
	Stream    bool      `json:"stream,omitempty"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Content string `json:"content"`
	Done    bool   `json:"done,omitempty"`
}