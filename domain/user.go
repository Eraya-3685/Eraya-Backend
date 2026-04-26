package domain

import (
	"time"
)

type User struct {
	ID           int64     `json:"id" db:"id"`
	FullName     string    `json:"full_name" db:"full_name"`
	Email        string    `json:"email" db:"email"`
	Phone        *string   `json:"phone" db:"phone"`
	PasswordHash string    `json:"-" db:"password_hash"`
	SocialID     *string   `json:"social_id" db:"social_id"`
	Role         string    `json:"role" db:"role"`
	Address      *string   `json:"address" db:"address"`
	AvatarURL    *string   `json:"avatar_url" db:"avatar_url"`
	Permissions  []string  `json:"permissions" db:"-"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	IsActive     bool      `json:"is_active" db:"is_active"`
}
