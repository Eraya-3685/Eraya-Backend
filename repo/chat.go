package repo

import (
	"context"
	"eraya/chat"
	"eraya/domain"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type chatRepo struct {
	db *sqlx.DB
}

func NewChatRepo(db *sqlx.DB) chat.ChatRepo {
	return &chatRepo{db: db}
}

func (r *chatRepo) FindOrCreateConversation(ctx context.Context, user1, user2 int64) (*domain.Conversation, error) {
	buyerID, adminID := user1, user2
	if user1 > user2 {
		buyerID, adminID = user2, user1
	}

	query := `SELECT * FROM conversations WHERE buyer_id = $1 AND admin_id = $2 LIMIT 1`
	var conv domain.Conversation
	err := r.db.GetContext(ctx, &conv, query, buyerID, adminID)

	if err == nil {
		return &conv, nil
	}

	insertQuery := `INSERT INTO conversations (buyer_id, admin_id) VALUES ($1, $2) RETURNING id, updated_at`
	err = r.db.QueryRowContext(ctx, insertQuery, buyerID, adminID).Scan(&conv.ID, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	conv.BuyerID = buyerID
	adminIDPtr := adminID
	conv.AdminID = &adminIDPtr
	return &conv, nil
}

func (r *chatRepo) SaveMessage(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	query := `
		INSERT INTO messages (conversation_id, sender_id, message_text)
		VALUES (:conversation_id, :sender_id, :message_text)
		RETURNING id, created_at
	`
	rows, err := r.db.NamedQueryContext(ctx, query, msg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		rows.Scan(&msg.ID, &msg.CreatedAt)
	}
	return msg, nil
}

func (r *chatRepo) GetConversationMessages(ctx context.Context, convID int64) ([]*domain.Message, error) {
	query := `SELECT * FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC`
	var messages []*domain.Message
	err := r.db.SelectContext(ctx, &messages, query, convID)
	return messages, err
}
