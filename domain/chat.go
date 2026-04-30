package domain

import "time"

type Conversation struct {
	ID          int64     `json:"id" db:"id"`
	BuyerID     int64     `json:"buyer_id" db:"buyer_id"`
	BuyerName   string    `json:"buyer_name" db:"buyer_name"`
	BuyerAvatar *string   `json:"buyer_avatar" db:"buyer_avatar"`
	AdminID     *int64    `json:"admin_id" db:"admin_id"`
	AdminName   *string   `json:"admin_name" db:"admin_name"`
	LastMessage *string   `json:"last_message" db:"last_message"`
	UnreadCount int       `json:"unread_count" db:"unread_count"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	Messages []Message `json:"messages,omitempty" db:"-"`
}

type Message struct {
	ID             int64     `json:"id" db:"id"`
	Type           string    `json:"type" db:"-"` // "message", "presence", "update", "delete"
	ConversationID int64     `json:"conversation_id" db:"conversation_id"`
	SenderID       int64     `json:"sender_id" db:"sender_id"`
	SenderName     string    `json:"sender_name" db:"sender_name"`
	SenderAvatar   *string   `json:"sender_avatar" db:"sender_avatar"`
	ReceiverID     int64     `json:"receiver_id" db:"-"`
	MessageText    *string   `json:"message_text" db:"message_text"`
	AttachmentURL  *string   `json:"attachment_url" db:"attachment_url"`
	IsRead         bool      `json:"is_read" db:"is_read"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	OrderID        *int64    `json:"order_id" db:"order_id"`
	ReplyToID      *int64    `json:"reply_to_id" db:"reply_to_id"`
	ReplyToText    *string   `json:"reply_to_text" db:"reply_to_text"`
	TempID         string    `json:"temp_id" db:"-"`
	Online         bool      `json:"online" db:"-"`
}
