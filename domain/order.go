package domain

import (
	"context"
	"time"
)

type Order struct {
	ID              int64      `json:"id" db:"id"`
	UserID          int64      `json:"user_id" db:"user_id"`
	TotalPrice      float64    `json:"total_price" db:"total_price"`
	PaymentMethod   string     `json:"payment_method" db:"payment_method"`
	PaymentStatus   string     `json:"payment_status" db:"payment_status"`
	OrderStatus     string     `json:"order_status" db:"order_status"`
	ShippingAddress string     `json:"shipping_address" db:"shipping_address"`
	TrxID           *string    `json:"trx_id" db:"trx_id"`
	SenderNumber    *string    `json:"sender_number" db:"sender_number"`
	PaidAmount      *float64   `json:"paid_amount" db:"paid_amount"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	TrackingNumber  *string    `json:"tracking_number" db:"tracking_number"`
	ConfirmedAt     *time.Time `json:"confirmed_at" db:"confirmed_at"`
	ProcessingAt    *time.Time `json:"processing_at" db:"processing_at"`
	ShippedAt       *time.Time `json:"shipped_at" db:"shipped_at"`
	DeliveredAt     *time.Time `json:"delivered_at" db:"delivered_at"`
	CouponCode      *string    `json:"coupon_code" db:"coupon_code"`
	DiscountAmount  float64    `json:"discount_amount" db:"discount_amount"`

	Items []OrderItem `json:"items,omitempty" db:"-"`
	User  User        `json:"user" db:"user"`
}

type OrderItem struct {
	ID              int64   `json:"id" db:"id"`
	OrderID         int64   `json:"order_id" db:"order_id"`
	ProductID       int64   `json:"product_id" db:"product_id"`
	Quantity        int     `json:"quantity" db:"quantity"`
	PriceAtPurchase float64 `json:"price_at_purchase" db:"price_at_purchase"`
	SelectedColor   string  `json:"selected_color" db:"selected_color"`
	SelectedSize    string  `json:"selected_size" db:"selected_size"`

	Product *Product `json:"product,omitempty" db:"-"`
}

type CartRepo interface {
	Add(ctx context.Context, item *CartItem) error
	List(ctx context.Context, userID int64) ([]*CartItem, error)
	Clear(ctx context.Context, userID int64) error
}

type OrderRepo interface {
	Create(ctx context.Context, order *Order, items []*OrderItem) (*Order, error)
	ListByUser(ctx context.Context, userID int64) ([]*Order, error)
	ListAll(ctx context.Context) ([]*Order, error)
	FindByID(ctx context.Context, id int64) (*Order, error)
	UpdateStatus(ctx context.Context, id int64, status, paymentStatus string) error
	UpdateStatusWithStock(ctx context.Context, id int64, status, paymentStatus string) error
	Delete(ctx context.Context, id int64) error
	GetCategorySales(ctx context.Context) ([]CategorySales, error)
	GetLowStockProducts(ctx context.Context) ([]Product, error)
}

type DashboardStats struct {
	TotalRevenue     float64         `json:"total_revenue"`
	TotalOrders      int             `json:"total_orders"`
	TotalSold        int             `json:"total_sold"`
	TotalProducts    int             `json:"total_products"`
	RevenueChart     []ChartData     `json:"revenue_chart"`
	VisitorChart     []ChartData     `json:"visitor_chart"`
	OrderStatusStats map[string]int  `json:"order_status_stats"`
	RecentMessages   []RecentMessage `json:"recent_messages"`
	RecentContacts   []RecentContact `json:"recent_contacts"`
	CategorySales    []CategorySales `json:"category_sales"`
	LowStockAlerts   []Product       `json:"low_stock_alerts"`
}

type ChartData struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	This  float64 `json:"this,omitempty"`
	Last  float64 `json:"last,omitempty"`
}

type RecentMessage struct {
	Name      string  `json:"name"`
	Msg       string  `json:"msg"`
	Time      string  `json:"time"`
	Unread    int     `json:"unread"`
	AvatarURL *string `json:"avatar_url"`
}

type RecentContact struct {
	ID        int64   `json:"id"`
	FullName  string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url"`
}

type CategorySales struct {
	CategoryName string  `json:"category_name" db:"category_name"`
	TotalSales   float64 `json:"total_sales" db:"total_sales"`
	ProductCount int     `json:"product_count" db:"product_count"`
}
