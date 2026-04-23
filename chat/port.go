package chat

import "eraya/domain"

type Service interface {
	SendMessage(senderID, receiverID int64, text string) (*domain.Message, error)
	GetConversation(userID1, userID2 int64) ([]*domain.Message, error)
}

type ChatRepo interface {
	SaveMessage(msg *domain.Message) (*domain.Message, error)
	GetConversationMessages(convID int64) ([]*domain.Message, error)
	FindOrCreateConversation(user1, user2 int64) (*domain.Conversation, error)
}

type ChatPubSub interface {
	PublishMessage(channel string, msg *domain.Message) error
	SubscribeToMessages(channel string, handler func(msg *domain.Message)) error
}
