package order

import "eraya/domain"

type Service interface {
	AddToCart(userID, productID int64, quantity int) error
	GetCart(userID int64) ([]*domain.CartItem, error)
	Checkout(userID int64, paymentMethod, shippingAddress string) (*domain.Order, error)
	GetOrders(userID int64) ([]*domain.Order, error)

	// Admin
	AdminGetAllOrders() ([]*domain.Order, error)
	AdminConfirmOrder(orderID int64) error
}

type CartRepo interface {
	Add(item *domain.CartItem) error
	List(userID int64) ([]*domain.CartItem, error)
	Clear(userID int64) error
}

type OrderRepo interface {
	Create(order *domain.Order, items []*domain.OrderItem) (*domain.Order, error)
	ListByUser(userID int64) ([]*domain.Order, error)
	ListAll() ([]*domain.Order, error)
	FindByID(id int64) (*domain.Order, error)
	UpdateStatus(id int64, status, paymentStatus string) error
}
