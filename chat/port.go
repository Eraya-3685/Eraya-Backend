package chat

import (
	"context"
	"eraya/domain"
)

type Service interface {
	SendMessage(ctx context.Context, senderID, receiverID int64, text string) (*domain.Message, error)
	GetConversation(ctx context.Context, userID1, userID2 int64) ([]*domain.Message, error)
}

type ChatRepo interface {
	SaveMessage(ctx context.Context, msg *domain.Message) (*domain.Message, error)
	GetConversationMessages(ctx context.Context, convID int64) ([]*domain.Message, error)
	FindOrCreateConversation(ctx context.Context, user1, user2 int64) (*domain.Conversation, error)
}

type ChatPubSub interface {
	PublishMessage(channel string, msg *domain.Message) error
	SubscribeToMessages(channel string, handler func(msg *domain.Message)) error
}
