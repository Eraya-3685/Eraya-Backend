package chat

import (
	"context"
	"eraya/domain"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type service struct {
	repo        ChatRepo
	pubsub      ChatPubSub
	onlineUsers sync.Map // map[int64]bool
}

func NewService(repo ChatRepo, pubsub ChatPubSub) Service {
	return &service{
		repo:   repo,
		pubsub: pubsub,
	}
}

func (s *service) SendMessage(ctx context.Context, senderID, receiverID int64, text string, replyToID *int64, tempID string) (*domain.Message, error) {
	// If receiver is 0, it might be an admin replying in a general view.
	// We should try to find the correct receiver from context or conversation if possible.
	// But usually, the handler should provide the correct withID.

	// Permission Check: If moderator, must have 'chat' permission
	hasChatAccess, _ := s.repo.HasPermission(ctx, senderID, "chat")
	if !hasChatAccess {
		// Only Buyers can send messages without 'chat' permission (since they are the ones seeking support)
		role, _ := s.repo.GetUserRole(ctx, senderID)
		if role == "moderator" {
			return nil, fmt.Errorf("unauthorized to send messages")
		}
	}

	conv, err := s.repo.FindOrCreateConversation(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}

	// Optimization: If receiverID was 0 but we found/created a conversation,
	// the actual receiver is the other person in the conversation.
	actualReceiverID := receiverID
	if receiverID == 0 || receiverID == senderID {
		if conv.BuyerID != senderID {
			actualReceiverID = conv.BuyerID
		} else if conv.AdminID != nil && *conv.AdminID != senderID {
			actualReceiverID = *conv.AdminID
		}
	}

	// Fetch sender name to include in real-time broadcast
	senderName, _ := s.repo.GetUserName(ctx, senderID)

	msg := &domain.Message{
		Type:           "message",
		ConversationID: conv.ID,
		SenderID:       senderID,
		SenderName:     senderName, // Include name here
		ReceiverID:     actualReceiverID,
		MessageText:    &text,
		ReplyToID:      replyToID,
		TempID:         tempID,
	}

	savedMsg, err := s.repo.SaveMessage(ctx, msg)
	if err != nil {
		return nil, err
	}
	savedMsg.TempID = tempID // Ensure it's in the returned object too
	savedMsg.Type = "message"

	// Publish to Redis with background goroutines and timeouts
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Always broadcast to the buyer's personal channel
		buyerChannel := fmt.Sprintf("chat_%d", conv.BuyerID)
		s.pubsub.PublishMessage(ctx, buyerChannel, savedMsg)

		// Always broadcast to the chat_admins channel
		s.pubsub.PublishMessage(ctx, "chat_admins", savedMsg)

		// If there is an assigned admin and it's not already covered by buyerChannel/chat_admins
		if conv.AdminID != nil {
			adminChannel := fmt.Sprintf("chat_%d", *conv.AdminID)
			if adminChannel != buyerChannel {
				s.pubsub.PublishMessage(ctx, adminChannel, savedMsg)
			}
		}
	}()

	return savedMsg, nil
}

func (s *service) GetConversation(ctx context.Context, userID1, userID2 int64) ([]*domain.Message, error) {
	conv, err := s.repo.FindOrCreateConversation(ctx, userID1, userID2)
	if err != nil {
		return nil, err
	}
	msgs, err := s.repo.GetMessages(ctx, conv.ID)
	if err != nil {
		return nil, err
	}
	for _, m := range msgs {
		m.Type = "message"
	}
	return msgs, nil
}

func (s *service) GetConversations(ctx context.Context, userID int64) ([]*domain.Conversation, error) {
	// Permission Check for Moderators
	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user role: %w", err)
	}
	if role == "moderator" {
		hasChatAccess, err := s.repo.HasPermission(ctx, userID, "chat")
		if err != nil {
			slog.Error("Permission check failed", "error", err, "userID", userID)
			return nil, fmt.Errorf("permission check failed")
		}
		if !hasChatAccess {
			return nil, fmt.Errorf("unauthorized: missing chat permission")
		}
	}
	return s.repo.ListConversations(ctx, userID)
}

func (s *service) Subscribe(ctx context.Context, userID, withID int64, handler func(msg *domain.Message)) error {
	personalChannel := fmt.Sprintf("chat_%d", userID)
	if err := s.pubsub.SubscribeToMessages(ctx, personalChannel, handler); err != nil {
		return err
	}

	// Subscribe to partner's presence updates
	if withID != 0 {
		presenceChannel := fmt.Sprintf("presence_%d", withID)
		if err := s.pubsub.SubscribeToMessages(ctx, presenceChannel, handler); err != nil {
			slog.Warn("Failed to subscribe to partner presence", "error", err, "partnerID", withID)
		}
	}

	// Permission Check for Admins/Moderators
	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil {
		if err != context.Canceled {
			slog.Error("Failed to get user role for subscription", "error", err, "userID", userID)
		}
		return nil
	}

	if role == "admin" {
		return s.pubsub.SubscribeToMessages(ctx, "chat_admins", handler)
	}

	if role == "moderator" {
		hasChatAccess, err := s.repo.HasPermission(ctx, userID, "chat")
		if err != nil {
			slog.Error("Permission check failed for moderator", "error", err, "userID", userID)
			return nil
		}
		if hasChatAccess {
			return s.pubsub.SubscribeToMessages(ctx, "chat_admins", handler)
		}
	}

	return nil
}

func (s *service) RegisterPresence(ctx context.Context, userID int64) {
	s.onlineUsers.Store(userID, true)
	s.broadcastPresence(userID, true)
}

func (s *service) UnregisterPresence(ctx context.Context, userID int64) {
	s.onlineUsers.Delete(userID)
	s.broadcastPresence(userID, false)
}

func (s *service) IsUserOnline(userID int64) bool {
	_, ok := s.onlineUsers.Load(userID)
	return ok
}

func (s *service) broadcastPresence(userID int64, online bool) {
	presenceMsg := &domain.Message{
		Type:     "presence",
		SenderID: userID,
		Online:   online,
	}
	// Broadcast to their own presence channel (for anyone chatting with them)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		presenceChannel := fmt.Sprintf("presence_%d", userID)
		s.pubsub.PublishMessage(ctx, presenceChannel, presenceMsg)
		
		// Also broadcast to admins for the sidebar list
		s.pubsub.PublishMessage(ctx, "chat_admins", presenceMsg)
	}()
}

func (s *service) UpdateMessage(ctx context.Context, userID, msgID int64, newText string) (*domain.Message, error) {
	msg, err := s.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}

	// Auth: Only sender can edit their own message
	if msg.SenderID != userID {
		return nil, fmt.Errorf("unauthorized to edit this message")
	}

	err = s.repo.UpdateMessage(ctx, msgID, newText)
	if err != nil {
		return nil, err
	}

	msg.MessageText = &newText
	msg.Type = "update"

	// Ensure receiver ID is set correctly for broadcast
	if msg.ReceiverID == 0 || msg.ReceiverID == userID {
		conv, err := s.repo.GetConversationByID(ctx, msg.ConversationID)
		if err == nil {
			if conv.BuyerID != userID {
				msg.ReceiverID = conv.BuyerID
			} else if conv.AdminID != nil {
				msg.ReceiverID = *conv.AdminID
			}
		}
	}

	// Broadcast update
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		receiverChannel := fmt.Sprintf("chat_%d", msg.ReceiverID)
		
		s.pubsub.PublishMessage(ctx, receiverChannel, msg)
		if receiverChannel != "chat_admins" {
			s.pubsub.PublishMessage(ctx, "chat_admins", msg)
		}
		s.pubsub.PublishMessage(ctx, fmt.Sprintf("chat_%d", userID), msg)
	}()

	return msg, nil
}

func (s *service) DeleteMessage(ctx context.Context, userID int64, msgID int64) error {
	msg, err := s.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return err
	}

	// Auth: Only sender can delete their own message
	if msg.SenderID != userID {
		return fmt.Errorf("unauthorized to delete this message")
	}

	err = s.repo.DeleteMessage(ctx, msgID)
	if err != nil {
		return err
	}

	msg.Type = "delete"

	// Ensure receiver ID is set correctly for broadcast
	if msg.ReceiverID == 0 || msg.ReceiverID == userID {
		conv, err := s.repo.GetConversationByID(ctx, msg.ConversationID)
		if err == nil {
			if conv.BuyerID != userID {
				msg.ReceiverID = conv.BuyerID
			} else if conv.AdminID != nil {
				msg.ReceiverID = *conv.AdminID
			}
		}
	}

	// Broadcast deletion
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		receiverChannel := fmt.Sprintf("chat_%d", msg.ReceiverID)
		
		s.pubsub.PublishMessage(ctx, receiverChannel, msg)
		if receiverChannel != "chat_admins" {
			s.pubsub.PublishMessage(ctx, "chat_admins", msg)
		}
		s.pubsub.PublishMessage(ctx, fmt.Sprintf("chat_%d", userID), msg)
	}()

	return nil
}

func (s *service) MarkAsRead(ctx context.Context, convID, userID int64) error {
	return s.repo.MarkAsRead(ctx, convID, userID)
}

func (s *service) IsAdmin(ctx context.Context, userID int64) (string, error) {
	return s.repo.GetUserRole(ctx, userID)
}

func (s *service) DeleteMessages(ctx context.Context, userID int64, msgIDs []int64) error {
	for _, id := range msgIDs {
		msg, err := s.repo.GetMessageByID(ctx, id)
		if err != nil {
			continue
		}

		if msg.SenderID != userID {
			continue
		}

		err = s.repo.DeleteMessage(ctx, id)
		if err != nil {
			continue
		}

		msg.Type = "delete"
		
		// Determine receiver for broadcast
		if msg.ReceiverID == 0 || msg.ReceiverID == userID {
			conv, err := s.repo.GetConversationByID(ctx, msg.ConversationID)
			if err == nil {
				if conv.BuyerID != userID {
					msg.ReceiverID = conv.BuyerID
				} else if conv.AdminID != nil {
					msg.ReceiverID = *conv.AdminID
				}
			}
		}

		// Broadcast to specific user and all admins
		go func(m *domain.Message) {
			bctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			receiverChannel := fmt.Sprintf("chat_%d", m.ReceiverID)
			s.pubsub.PublishMessage(bctx, receiverChannel, m)
			if receiverChannel != "chat_admins" {
				s.pubsub.PublishMessage(bctx, "chat_admins", m)
			}
			s.pubsub.PublishMessage(bctx, fmt.Sprintf("chat_%d", userID), m)
		}(msg)
	}

	return nil
}

func (s *service) DeleteConversation(ctx context.Context, userID, convID int64) error {
	// Auth: Only admins can delete conversations
	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil || (role != "admin" && role != "moderator") {
		return fmt.Errorf("unauthorized to delete conversation")
	}

	// Get conversation details before deleting to know the buyerID
	conv, err := s.repo.GetConversationByID(ctx, convID)
	if err != nil {
		return fmt.Errorf("conversation not found")
	}
	targetBuyerID := conv.BuyerID

	err = s.repo.DeleteConversation(ctx, convID)
	if err != nil {
		return err
	}

	// Broadcast deletion to buyer and admins
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		msg := &domain.Message{
			Type:           "delete_conversation",
			ConversationID: convID,
		}
		if targetBuyerID != 0 {
			s.pubsub.PublishMessage(ctx, fmt.Sprintf("chat_%d", targetBuyerID), msg)
		}
		s.pubsub.PublishMessage(ctx, "chat_admins", msg)
	}()

	return nil
}

func (s *service) SearchUsers(ctx context.Context, query string) ([]*domain.User, error) {
	return s.repo.SearchUsers(ctx, query)
}
func (s *service) GetTotalUnreadCount(ctx context.Context, userID int64) (int, error) {
	return s.repo.GetTotalUnreadCount(ctx, userID)
}
