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
	if user2 == 0 || user1 == user2 {
		u, err := r.fetchUserMinimal(ctx, user1)
		if err != nil {
			return nil, fmt.Errorf("user not found: %v", err)
		}
		if u.Role == "admin" || u.Role == "moderator" {
			return nil, fmt.Errorf("admins cannot initiate a support conversation for themselves")
		}
		return r.getOrCreateBuyerConversation(ctx, user1)
	}

	var users []userMinimal
	query := `SELECT id, full_name, role, avatar_url FROM users WHERE id IN ($1, $2)`
	err := r.db.SelectContext(ctx, &users, query, user1, user2)
	if err != nil {
		return nil, err
	}

	var u1, u2 userMinimal
	for _, u := range users {
		if u.ID == user1 {
			u1 = u
		}
		if u.ID == user2 {
			u2 = u
		}
	}

	if u1.ID == 0 {
		return nil, fmt.Errorf("user %d not found", user1)
	}
	if u2.ID == 0 {
		return nil, fmt.Errorf("user %d not found", user2)
	}

	var buyerID int64
	var adminID *int64
	var buyerName string
	var adminName *string

	if u1.Role == "admin" || u1.Role == "moderator" {
		adminID = &u1.ID
		adminName = &u1.FullName
		buyerID = u2.ID
		buyerName = u2.FullName
	} else {
		buyerID = u1.ID
		buyerName = u1.FullName
		if u2.Role == "admin" || u2.Role == "moderator" {
			adminID = &u2.ID
			adminName = &u2.FullName
		}
	}

	if adminID == nil {
		return r.getOrCreateBuyerConversation(ctx, buyerID)
	}

	var existingConv domain.Conversation
	checkQuery := `SELECT * FROM conversations WHERE buyer_id = $1 LIMIT 1`
	err = r.db.GetContext(ctx, &existingConv, checkQuery, buyerID)

	if err == nil {
		if existingConv.AdminID == nil || *existingConv.AdminID != *adminID {
			updateQuery := `UPDATE conversations SET admin_id = $1 WHERE id = $2`
			_, _ = r.db.ExecContext(ctx, updateQuery, *adminID, existingConv.ID)
			existingConv.AdminID = adminID
		}
		existingConv.BuyerName = buyerName
		existingConv.AdminName = adminName
		return &existingConv, nil
	}

	insertQuery := `INSERT INTO conversations (buyer_id, admin_id) VALUES ($1, $2) RETURNING *`
	var newConv domain.Conversation
	err = r.db.GetContext(ctx, &newConv, insertQuery, buyerID, *adminID)
	if err != nil {
		return nil, err
	}
	
	newConv.BuyerName = buyerName
	newConv.AdminName = adminName
	return &newConv, nil
}

func (r *chatRepo) getOrCreateBuyerConversation(ctx context.Context, buyerID int64) (*domain.Conversation, error) {
	query := `
		SELECT c.*, u.full_name as buyer_name 
		FROM conversations c 
		JOIN users u ON c.buyer_id = u.id 
		WHERE c.buyer_id = $1 
		LIMIT 1`
	var conv domain.Conversation
	err := r.db.GetContext(ctx, &conv, query, buyerID)
	if err == nil {
		return &conv, nil
	}

	insertQuery := `
		INSERT INTO conversations (buyer_id, admin_id) 
		VALUES ($1, NULL) 
		RETURNING id, updated_at`
	err = r.db.QueryRowContext(ctx, insertQuery, buyerID).Scan(&conv.ID, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create unassigned conversation: %w", err)
	}

	u, _ := r.fetchUserMinimal(ctx, buyerID)
	conv.BuyerName = u.FullName
	conv.BuyerID = buyerID
	conv.AdminID = nil
	return &conv, nil
}

func (r *chatRepo) SaveMessage(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	sender, err := r.fetchUserMinimal(ctx, msg.SenderID)
	if err != nil {
		return nil, err
	}

	if sender.Role == "admin" || sender.Role == "moderator" {
		assignQuery := `UPDATE conversations SET admin_id = $1 WHERE id = $2 AND admin_id IS NULL`
		r.db.ExecContext(ctx, assignQuery, msg.SenderID, msg.ConversationID)
	}

	query := `
		INSERT INTO messages (conversation_id, sender_id, message_text, reply_to_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err = r.db.QueryRowContext(ctx, query, msg.ConversationID, msg.SenderID, msg.MessageText, msg.ReplyToID).
		Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	updateConvQuery := `UPDATE conversations SET last_message = $1, updated_at = NOW() WHERE id = $2`
	_, _ = r.db.ExecContext(ctx, updateConvQuery, msg.MessageText, msg.ConversationID)

	msg.SenderName = sender.FullName
	msg.SenderAvatar = sender.AvatarURL

	return msg, nil
}

func (r *chatRepo) GetMessages(ctx context.Context, convID int64) ([]*domain.Message, error) {
	query := `
		SELECT m.*, u.full_name as sender_name, u.avatar_url as sender_avatar, rm.message_text as reply_to_text
		FROM messages m 
		JOIN users u ON m.sender_id = u.id 
		LEFT JOIN messages rm ON m.reply_to_id = rm.id
		WHERE m.conversation_id = $1 
		ORDER BY m.created_at ASC`
	var messages []*domain.Message
	err := r.db.SelectContext(ctx, &messages, query, convID)
	return messages, err
}

func (r *chatRepo) ListConversations(ctx context.Context, userID int64) ([]*domain.Conversation, error) {
	user, err := r.fetchUserMinimal(ctx, userID)
	if err != nil {
		return nil, err
	}

	condition := "c.buyer_id = $1"
	if user.Role == "admin" || user.Role == "moderator" {
		condition = "c.admin_id = $1 OR c.admin_id IS NULL"
	}

	query := fmt.Sprintf(`
		SELECT * FROM (
			SELECT DISTINCT ON (c.buyer_id) c.*, 
				b.full_name as buyer_name, 
				b.avatar_url as buyer_avatar, 
				a.full_name as admin_name,
				(SELECT COUNT(*) FROM messages m WHERE m.conversation_id = c.id AND m.is_read = false AND m.sender_id != $1) as unread_count
			FROM conversations c
			JOIN users b ON c.buyer_id = b.id
			LEFT JOIN users a ON c.admin_id = a.id
			WHERE %s
			ORDER BY c.buyer_id, c.updated_at DESC
		) t ORDER BY updated_at DESC`, condition)

	var convs []*domain.Conversation
	err = r.db.SelectContext(ctx, &convs, query, userID)
	return convs, err
}

func (r *chatRepo) GetUserRole(ctx context.Context, userID int64) (string, error) {
	var role string
	err := r.db.GetContext(ctx, &role, "SELECT role FROM users WHERE id = $1", userID)
	return role, err
}

func (r *chatRepo) GetMessageByID(ctx context.Context, msgID int64) (*domain.Message, error) {
	var msg domain.Message
	query := `
		SELECT m.*, u.full_name as sender_name, u.avatar_url as sender_avatar, rm.message_text as reply_to_text
		FROM messages m 
		JOIN users u ON m.sender_id = u.id 
		LEFT JOIN messages rm ON m.reply_to_id = rm.id
		WHERE m.id = $1`
	err := r.db.GetContext(ctx, &msg, query, msgID)
	return &msg, err
}

func (r *chatRepo) UpdateMessage(ctx context.Context, msgID int64, newText string) error {
	// 1. Update the message itself
	query := "UPDATE messages SET message_text = $1 WHERE id = $2"
	_, err := r.db.ExecContext(ctx, query, newText, msgID)
	if err != nil {
		return err
	}

	// 2. If this was the last message in the conversation, update the preview text
	// but DON'T change updated_at so it doesn't jump to top
	updateConvQuery := `
		UPDATE conversations 
		SET last_message = $1 
		WHERE id = (SELECT conversation_id FROM messages WHERE id = $2)
		AND id IN (
			SELECT conversation_id FROM messages 
			GROUP BY conversation_id 
			HAVING MAX(id) = $2
		)`
	_, _ = r.db.ExecContext(ctx, updateConvQuery, newText, msgID)
	
	return nil
}

func (r *chatRepo) DeleteMessage(ctx context.Context, msgID int64) error {
	query := "DELETE FROM messages WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, msgID)
	return err
}

func (r *chatRepo) DeleteMessages(ctx context.Context, msgIDs []int64) error {
	query, args, err := sqlx.In("DELETE FROM messages WHERE id IN (?)", msgIDs)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)
	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *chatRepo) DeleteConversation(ctx context.Context, convID int64) error {
	_, _ = r.db.ExecContext(ctx, "DELETE FROM messages WHERE conversation_id = $1", convID)
	_, err := r.db.ExecContext(ctx, "DELETE FROM conversations WHERE id = $1", convID)
	return err
}

func (r *chatRepo) MarkAsRead(ctx context.Context, convID, userID int64) error {
	query := `UPDATE messages SET is_read = true WHERE conversation_id = $1 AND sender_id != $2 AND is_read = false`
	_, err := r.db.ExecContext(ctx, query, convID, userID)
	return err
}

func (r *chatRepo) GetUserName(ctx context.Context, userID int64) (string, error) {
	var name string
	err := r.db.GetContext(ctx, &name, "SELECT full_name FROM users WHERE id = $1", userID)
	return name, err
}

func (r *chatRepo) SearchUsers(ctx context.Context, query string) ([]*domain.User, error) {
	var users []*domain.User
	q := `SELECT id, full_name, role, avatar_url FROM users 
	      WHERE (full_name ILIKE $1 OR id::text = $2)
	      LIMIT 20`
	err := r.db.SelectContext(ctx, &users, q, "%"+query+"%", query)
	return users, err
}

type userMinimal struct {
	ID        int64   `db:"id"`
	FullName  string  `db:"full_name"`
	Role      string  `db:"role"`
	AvatarURL *string `db:"avatar_url"`
}

func (r *chatRepo) fetchUserMinimal(ctx context.Context, id int64) (userMinimal, error) {
	var u userMinimal
	err := r.db.GetContext(ctx, &u, "SELECT id, full_name, role, avatar_url FROM users WHERE id = $1", id)
	return u, err
}

func (r *chatRepo) GetConversationByID(ctx context.Context, convID int64) (*domain.Conversation, error) {
	var conv domain.Conversation
	query := "SELECT * FROM conversations WHERE id = $1"
	err := r.db.GetContext(ctx, &conv, query, convID)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}
