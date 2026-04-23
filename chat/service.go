package chat

import (
	"context"
	"eraya/domain"
	"fmt"
)

type service struct {
	repo   ChatRepo
	pubsub ChatPubSub
}

func NewService(repo ChatRepo, pubsub ChatPubSub) Service {
	return &service{
		repo:   repo,
		pubsub: pubsub,
	}
}

func (s *service) SendMessage(ctx context.Context, senderID, receiverID int64, text string) (*domain.Message, error) {
	conv, err := s.repo.FindOrCreateConversation(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}

	msg := &domain.Message{
		ConversationID: conv.ID,
		SenderID:       senderID,
		MessageText:    &text,
	}

	savedMsg, err := s.repo.SaveMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Publish to Redis so other WebSocket instances can broadcast it
	channel := fmt.Sprintf("chat_%d", receiverID)
	go s.pubsub.PublishMessage(channel, savedMsg)

	return savedMsg, nil
}

func (s *service) GetConversation(ctx context.Context, userID1, userID2 int64) ([]*domain.Message, error) {
	conv, err := s.repo.FindOrCreateConversation(ctx, userID1, userID2)
	if err != nil {
		return nil, err
	}
	return s.repo.GetConversationMessages(ctx, conv.ID)
}
