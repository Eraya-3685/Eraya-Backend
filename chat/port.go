package chat

import (
	"context"
	"eraya/domain"
)

type Service interface {
	SendMessage(ctx context.Context, senderID, receiverID int64, text string, replyToID *int64, tempID string) (*domain.Message, error)
	GetConversation(ctx context.Context, userID1, userID2 int64) ([]*domain.Message, error)
	GetConversations(ctx context.Context, userID int64) ([]*domain.Conversation, error)
	Subscribe(ctx context.Context, userID, withID int64, handler func(msg *domain.Message)) error
	UpdateMessage(ctx context.Context, userID, msgID int64, newText string) (*domain.Message, error)
	DeleteMessage(ctx context.Context, userID, msgID int64) error
	DeleteMessages(ctx context.Context, userID int64, msgIDs []int64) error
	DeleteConversation(ctx context.Context, userID, convID int64) error
	MarkAsRead(ctx context.Context, convID, userID int64) error
	IsAdmin(ctx context.Context, userID int64) (string, error)
	RegisterPresence(ctx context.Context, userID int64)
	UnregisterPresence(ctx context.Context, userID int64)
	IsUserOnline(userID int64) bool
	SearchUsers(ctx context.Context, query string) ([]*domain.User, error)
}

type ChatRepo interface {
	SaveMessage(ctx context.Context, msg *domain.Message) (*domain.Message, error)
	GetMessages(ctx context.Context, convID int64) ([]*domain.Message, error)
	ListConversations(ctx context.Context, userID int64) ([]*domain.Conversation, error)
	FindOrCreateConversation(ctx context.Context, user1, user2 int64) (*domain.Conversation, error)
	GetUserRole(ctx context.Context, userID int64) (string, error)
	UpdateMessage(ctx context.Context, msgID int64, newText string) error
	DeleteMessage(ctx context.Context, msgID int64) error
	DeleteMessages(ctx context.Context, msgIDs []int64) error
	DeleteConversation(ctx context.Context, convID int64) error
	GetMessageByID(ctx context.Context, msgID int64) (*domain.Message, error)
	MarkAsRead(ctx context.Context, convID, userID int64) error
	GetUserName(ctx context.Context, userID int64) (string, error)
	SearchUsers(ctx context.Context, query string) ([]*domain.User, error)
	GetConversationByID(ctx context.Context, convID int64) (*domain.Conversation, error)
}

type ChatPubSub interface {
	PublishMessage(ctx context.Context, channel string, msg *domain.Message) error
	SubscribeToMessages(ctx context.Context, channel string, handler func(msg *domain.Message)) error
}
