package domain

import "time"

type Review struct {
	ID         int64     `json:"id" db:"id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	ProductID  int64     `json:"product_id" db:"product_id"`
	Rating     int       `json:"rating" db:"rating"`
	Comment    *string   `json:"comment" db:"comment"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	IsVerified bool      `json:"is_verified" db:"is_verified"`
	IsApproved bool      `json:"is_approved" db:"is_approved"`
	ImageURL   *string   `json:"image_url" db:"image_url"`

	User *User `json:"user,omitempty" db:"user"`
}
