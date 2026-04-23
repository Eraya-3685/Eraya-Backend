package order

import (
	"eraya/domain"
	"eraya/product"
	"errors"
	"fmt"
	"log"
	"time"
)

type service struct {
	cartRepo   CartRepo
	orderRepo  OrderRepo
	productSvc product.Service
}

func NewService(cartRepo CartRepo, orderRepo OrderRepo, productSvc product.Service) Service {
	return &service{
		cartRepo:   cartRepo,
		orderRepo:  orderRepo,
		productSvc: productSvc,
	}
}

func (s *service) AddToCart(userID, productID int64, quantity int) error {
	item := &domain.CartItem{
		UserID:    userID,
		ProductID: productID,
		Quantity:  quantity,
	}
	return s.cartRepo.Add(item)
}

func (s *service) GetCart(userID int64) ([]*domain.CartItem, error) {
	return s.cartRepo.List(userID)
}

func (s *service) Checkout(userID int64, paymentMethod, shippingAddress string) (*domain.Order, error) {
	cartItems, err := s.cartRepo.List(userID)
	if err != nil || len(cartItems) == 0 {
		return nil, errors.New("cart is empty")
	}

	var total float64
	var orderItems []*domain.OrderItem

	for _, item := range cartItems {
		p, err := s.productSvc.GetProductByID(item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product not found: %d", item.ProductID)
		}

		price := p.BasePrice
		if p.DiscountPrice != nil && *p.DiscountPrice > 0 {
			price = *p.DiscountPrice
		}

		total += price * float64(item.Quantity)
		orderItems = append(orderItems, &domain.OrderItem{
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			PriceAtPurchase: price,
		})
	}

	order := &domain.Order{
		UserID:          userID,
		TotalPrice:      total,
		PaymentMethod:   paymentMethod,
		PaymentStatus:   "pending",
		OrderStatus:     "pending",
		ShippingAddress: shippingAddress,
	}

	createdOrder, err := s.orderRepo.Create(order, orderItems)
	if err != nil {
		return nil, err
	}

	// Clear cart after successful order
	s.cartRepo.Clear(userID)

	if paymentMethod == "bKash" {
		// Auto confirm
		s.AdminConfirmOrder(createdOrder.ID)
	}

	return createdOrder, nil
}

func (s *service) GetOrders(userID int64) ([]*domain.Order, error) {
	return s.orderRepo.ListByUser(userID)
}

func (s *service) AdminGetAllOrders() ([]*domain.Order, error) {
	return s.orderRepo.ListAll()
}

func (s *service) AdminConfirmOrder(orderID int64) error {
	err := s.orderRepo.UpdateStatus(orderID, "confirmed", "paid") // If COD, confirmed means payment is expected later, but we simplify here.
	if err == nil {
		// Async notifications
		go func() {
			time.Sleep(1 * time.Second) // Simulate network delay
			log.Printf("Email and SMS sent to user for Order ID: %d", orderID)
		}()
	}
	return err
}
