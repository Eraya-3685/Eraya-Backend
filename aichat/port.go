package aichat

import "context"

// ChatMessage represents a single message in the conversation history.
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// Service defines the AI chat interface.
type Service interface {
	Chat(ctx context.Context, userMessage string, history []ChatMessage) (string, error)
}
