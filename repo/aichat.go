package repo

import (
	"context"
	"eraya/aichat"
	"time"

	"github.com/jmoiron/sqlx"
)

type aiChatRepo struct {
	db *sqlx.DB
}

func NewAIChatRepo(db *sqlx.DB) aichat.Repo {
	return &aiChatRepo{db: db}
}

func (r *aiChatRepo) SaveMessage(ctx context.Context, userID int64, role, content string) error {
	query := `
		INSERT INTO ai_chat_messages (user_id, role, content, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query, userID, role, content, time.Now())
	return err
}

func (r *aiChatRepo) GetHistory(ctx context.Context, userID int64, limit int) ([]aichat.ChatMessage, error) {
	query := `
		SELECT role, content
		FROM ai_chat_messages
		WHERE user_id = $1
		ORDER BY created_at ASC
	`
	// We sort ASC directly since we want chronological order for history
	// Wait, if we want the LAST 'limit' messages, we might need a subquery or limit first.
	// We'll get all for simplicity if limit isn't strict, but let's do a subquery.
	if limit > 0 {
		query = `
			SELECT role, content FROM (
				SELECT role, content, created_at
				FROM ai_chat_messages
				WHERE user_id = $1
				ORDER BY created_at DESC
				LIMIT $2
			) sub
			ORDER BY created_at ASC
		`
		var msgs []aichat.ChatMessage
		err := r.db.SelectContext(ctx, &msgs, query, userID, limit)
		return msgs, err
	}

	var msgs []aichat.ChatMessage
	err := r.db.SelectContext(ctx, &msgs, query, userID)
	return msgs, err
}
