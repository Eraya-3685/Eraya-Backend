package domain

import "time"

type Conversation struct {
	ID          int64     `json:"id" db:"id"`
	BuyerID     int64     `json:"buyer_id" db:"buyer_id"`
	AdminID     *int64    `json:"admin_id" db:"admin_id"`
	LastMessage *string   `json:"last_message" db:"last_message"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	Messages []Message `json:"messages,omitempty" db:"-"`
}

type Message struct {
	ID             int64     `json:"id" db:"id"`
	ConversationID int64     `json:"conversation_id" db:"conversation_id"`
	SenderID       int64     `json:"sender_id" db:"sender_id"`
	MessageText    *string   `json:"message_text" db:"message_text"`
	AttachmentURL  *string   `json:"attachment_url" db:"attachment_url"`
	IsRead         bool      `json:"is_read" db:"is_read"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	OrderID        *int64    `json:"order_id" db:"order_id"`
}
