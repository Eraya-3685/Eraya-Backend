package order

import (
	"context"
	"eraya/domain"
)

type Service interface {
	AddToCart(ctx context.Context, userID, productID int64, quantity int) error
	GetCart(ctx context.Context, userID int64) ([]*domain.CartItem, error)
	Checkout(ctx context.Context, userID int64, items []domain.CartItem, paymentMethod, shippingAddress string, trxID, senderNumber *string, paidAmount *float64) (*domain.Order, error)
	GetOrders(ctx context.Context, userID int64) ([]*domain.Order, error)
	GetOrderByID(ctx context.Context, orderID, userID int64) (*domain.Order, error)
	ConfirmPayment(ctx context.Context, orderID int64, trxID string, amount float64) error

	// Admin
	AdminGetAllOrders(ctx context.Context) ([]*domain.Order, error)
	AdminConfirmOrder(ctx context.Context, orderID int64) error
	AdminUpdateOrderStatus(ctx context.Context, orderID int64, status string, estimatedDate string) error
	AdminRequestDeleteOTP(ctx context.Context, adminID int64) error
	AdminDeleteOrder(ctx context.Context, id int64, otp string, adminID int64) error
	AdminGetDashboardStats(ctx context.Context) (*domain.DashboardStats, error)
}

type CartRepo interface {
	Add(ctx context.Context, item *domain.CartItem) error
	List(ctx context.Context, userID int64) ([]*domain.CartItem, error)
	Clear(ctx context.Context, userID int64) error
}

type OrderRepo interface {
	Create(ctx context.Context, order *domain.Order, items []*domain.OrderItem) (*domain.Order, error)
	ListByUser(ctx context.Context, userID int64) ([]*domain.Order, error)
	ListAll(ctx context.Context) ([]*domain.Order, error)
	FindByID(ctx context.Context, id int64) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id int64, status, paymentStatus string) error
	Delete(ctx context.Context, id int64) error
}
