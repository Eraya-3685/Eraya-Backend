package domain

type CartItem struct {
	ID            int64    `json:"id" db:"id"`
	UserID        int64    `json:"user_id" db:"user_id"`
	ProductID     int64    `json:"product_id" db:"product_id"`
	Quantity      int      `json:"quantity" db:"quantity"`
	SelectedColor string   `json:"selected_color" db:"selected_color"`
	SelectedSize  string   `json:"selected_size" db:"selected_size"`

	Product *Product `json:"product,omitempty" db:"product"`
}
