package domain

import "time"

type Wishlist struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	ProductID int64     `json:"product_id" db:"product_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	Product *Product `json:"product,omitempty" db:"-"`
}
