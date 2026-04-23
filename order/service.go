package order

import (
	"context"
	"eraya/domain"
	"eraya/product"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
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

func (s *service) AddToCart(ctx context.Context, userID, productID int64, quantity int) error {
	item := &domain.CartItem{
		UserID:    userID,
		ProductID: productID,
		Quantity:  quantity,
	}
	return s.cartRepo.Add(ctx, item)
}

func (s *service) GetCart(ctx context.Context, userID int64) ([]*domain.CartItem, error) {
	return s.cartRepo.List(ctx, userID)
}

func (s *service) Checkout(ctx context.Context, userID int64, paymentMethod, shippingAddress string) (*domain.Order, error) {
	cartItems, err := s.cartRepo.List(ctx, userID)
	if err != nil || len(cartItems) == 0 {
		return nil, errors.New("cart is empty")
	}

	var total float64
	var orderItems []*domain.OrderItem
	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)

	for _, item := range cartItems {
		item := item // Capture for closure
		g.Go(func() error {
			p, err := s.productSvc.GetProductByID(gCtx, item.ProductID)
			if err != nil {
				return fmt.Errorf("product not found: %d", item.ProductID)
			}

			price := p.BasePrice
			if p.DiscountPrice != nil && *p.DiscountPrice > 0 {
				price = *p.DiscountPrice
			}

			mu.Lock()
			total += price * float64(item.Quantity)
			orderItems = append(orderItems, &domain.OrderItem{
				ProductID:       item.ProductID,
				Quantity:        item.Quantity,
				PriceAtPurchase: price,
			})
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	order := &domain.Order{
		UserID:          userID,
		TotalPrice:      total,
		PaymentMethod:   paymentMethod,
		PaymentStatus:   "pending",
		OrderStatus:     "pending",
		ShippingAddress: shippingAddress,
	}

	createdOrder, err := s.orderRepo.Create(ctx, order, orderItems)
	if err != nil {
		return nil, err
	}

	// Clear cart after successful order
	s.cartRepo.Clear(ctx, userID)

	if paymentMethod == "bKash" {
		// Auto confirm
		s.AdminConfirmOrder(ctx, createdOrder.ID)
	}

	return createdOrder, nil
}

func (s *service) GetOrders(ctx context.Context, userID int64) ([]*domain.Order, error) {
	return s.orderRepo.ListByUser(ctx, userID)
}

func (s *service) AdminGetAllOrders(ctx context.Context) ([]*domain.Order, error) {
	return s.orderRepo.ListAll(ctx)
}

func (s *service) AdminConfirmOrder(ctx context.Context, orderID int64) error {
	err := s.orderRepo.UpdateStatus(ctx, orderID, "confirmed", "paid")
	if err == nil {
		// Async notifications
		go func() {
			time.Sleep(1 * time.Second)
			slog.Info("Email and SMS sent to user", "order_id", orderID)
		}()
	}
	return err
}

func (s *service) AdminDeleteOrder(ctx context.Context, id int64) error {
	return s.orderRepo.Delete(ctx, id)
}
