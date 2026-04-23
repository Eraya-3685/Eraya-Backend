package repo

import (
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

func (r *chatRepo) FindOrCreateConversation(user1, user2 int64) (*domain.Conversation, error) {
	// Simple assumption: user1 is buyer, user2 is admin, or vice versa.
	// For simplicity, we just use buyer_id and admin_id without checking roles here, but in reality we should order them.
	buyerID, adminID := user1, user2
	if user1 > user2 { // Simple stable sorting to always find the same row
		buyerID, adminID = user2, user1
	}

	query := `SELECT * FROM conversations WHERE buyer_id = $1 AND admin_id = $2 LIMIT 1`
	var conv domain.Conversation
	err := r.db.Get(&conv, query, buyerID, adminID)
	
	if err == nil {
		return &conv, nil
	}

	// Create new
	insertQuery := `INSERT INTO conversations (buyer_id, admin_id) VALUES ($1, $2) RETURNING id, updated_at`
	err = r.db.QueryRow(insertQuery, buyerID, adminID).Scan(&conv.ID, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	conv.BuyerID = buyerID
	
	adminIDPtr := adminID
	conv.AdminID = &adminIDPtr
	return &conv, nil
}

func (r *chatRepo) SaveMessage(msg *domain.Message) (*domain.Message, error) {
	query := `
		INSERT INTO messages (conversation_id, sender_id, receiver_id, text)
		VALUES (:conversation_id, :sender_id, :receiver_id, :text)
		RETURNING id, created_at
	`
	rows, err := r.db.NamedQuery(query, msg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		rows.Scan(&msg.ID, &msg.CreatedAt, &msg.IsRead)
	}
	return msg, nil
}

func (r *chatRepo) GetConversationMessages(convID int64) ([]*domain.Message, error) {
	query := `SELECT * FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC`
	var messages []*domain.Message
	err := r.db.Select(&messages, query, convID)
	return messages, err
}
