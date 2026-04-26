package order

import (
	"context"
	"eraya/domain"
)

type Service interface {
	AddToCart(ctx context.Context, userID, productID int64, quantity int) error
	GetCart(ctx context.Context, userID int64) ([]*domain.CartItem, error)
	Checkout(ctx context.Context, userID int64, items []domain.CartItem, paymentMethod, shippingAddress string) (*domain.Order, error)
	GetOrders(ctx context.Context, userID int64) ([]*domain.Order, error)

	// Admin
	AdminGetAllOrders(ctx context.Context) ([]*domain.Order, error)
	AdminConfirmOrder(ctx context.Context, orderID int64) error
	AdminDeleteOrder(ctx context.Context, id int64) error
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
