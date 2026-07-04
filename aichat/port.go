package aichat

import "context"

// ChatMessage represents a single message in the conversation history.
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// Repo defines the persistence layer for AI chat messages.
type Repo interface {
	SaveMessage(ctx context.Context, userID int64, role, content string) error
	GetHistory(ctx context.Context, userID int64, limit int) ([]ChatMessage, error)
}

// Service defines the AI chat interface.
type Service interface {
	Chat(ctx context.Context, userID *int64, userMessage string, history []ChatMessage) (string, error)
	GetHistory(ctx context.Context, userID int64, limit int) ([]ChatMessage, error)
}
