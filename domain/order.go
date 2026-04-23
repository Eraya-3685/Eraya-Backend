package domain

import "time"

type Order struct {
	ID              int64      `json:"id" db:"id"`
	UserID          int64      `json:"user_id" db:"user_id"`
	TotalPrice      float64    `json:"total_price" db:"total_price"`
	PaymentMethod   string     `json:"payment_method" db:"payment_method"`
	PaymentStatus   string     `json:"payment_status" db:"payment_status"`
	OrderStatus     string     `json:"order_status" db:"order_status"`
	ShippingAddress string     `json:"shipping_address" db:"shipping_address"`
	TrxID           *string    `json:"trx_id" db:"trx_id"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	TrackingNumber  *string    `json:"tracking_number" db:"tracking_number"`
	ConfirmedAt     *time.Time `json:"confirmed_at" db:"confirmed_at"`
	ShippedAt       *time.Time `json:"shipped_at" db:"shipped_at"`
	DeliveredAt     *time.Time `json:"delivered_at" db:"delivered_at"`

	Items []OrderItem `json:"items,omitempty" db:"-"`
}

type OrderItem struct {
	ID              int64   `json:"id" db:"id"`
	OrderID         int64   `json:"order_id" db:"order_id"`
	ProductID       int64   `json:"product_id" db:"product_id"`
	Quantity        int     `json:"quantity" db:"quantity"`
	PriceAtPurchase float64 `json:"price_at_purchase" db:"price_at_purchase"`

	Product *Product `json:"product,omitempty" db:"-"`
}
