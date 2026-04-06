package agent

import (
	"context"
	"fmt"

	"backend/internal/pkg/llm"
)

// baseAgent provides common functionality for all agents
type baseAgent struct {
	name        string
	role        string
	description string
	prompt      string
	llmClient   llm.Client
}

// Name returns the agent's name
func (a *baseAgent) Name() string {
	return a.name
}

// Role returns the agent's role
func (a *baseAgent) Role() string {
	return a.role
}

// Description returns the agent's description
func (a *baseAgent) Description() string {
	return a.description
}

// GetPrompt returns the agent's prompt
func (a *baseAgent) GetPrompt() string {
	return a.prompt
}

// SetPrompt updates the agent's prompt
func (a *baseAgent) SetPrompt(prompt string) {
	a.prompt = prompt
}

// SetLLMClient sets the LLM client for the agent
func (a *baseAgent) SetLLMClient(client llm.Client) {
	a.llmClient = client
}

// CallLLM calls the LLM with the given input
func (a *baseAgent) CallLLM(input string) (string, error) {
	if a.llmClient == nil {
		return "", fmt.Errorf("LLM client not set")
	}
	opts := llm.CompletionOptions{
		Temperature: 0.7,
		MaxTokens:   4096,
	}
	return a.llmClient.Complete(context.Background(), input, opts)
}
