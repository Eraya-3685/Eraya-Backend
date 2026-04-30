package order

import (
	"context"
	"eraya/domain"
	"eraya/infra/mail"
	"eraya/product"
	"eraya/settings"
	"eraya/user"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"
	"strings"

	"golang.org/x/sync/errgroup"
)

type service struct {
	cartRepo    domain.CartRepo
	orderRepo   domain.OrderRepo
	productSvc  product.Service
	settingsSvc settings.Service
	mailer      mail.Mailer
	userSvc     user.Service
}

func NewService(cartRepo domain.CartRepo, orderRepo domain.OrderRepo, productSvc product.Service, settingsSvc settings.Service, mailer mail.Mailer, userSvc user.Service) Service {
	return &service{
		cartRepo:    cartRepo,
		orderRepo:   orderRepo,
		productSvc:  productSvc,
		settingsSvc: settingsSvc,
		mailer:      mailer,
		userSvc:     userSvc,
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

func (s *service) Checkout(ctx context.Context, userID int64, items []domain.CartItem, paymentMethod, shippingAddress string, trxID, senderNumber *string, paidAmount *float64) (*domain.Order, error) {
	if len(items) == 0 {
		return nil, errors.New("order must have items")
	}

	cartItems := items

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

			// 1. Check stock
			if p.StockCount < item.Quantity {
				return fmt.Errorf("insufficient stock for product: %s", p.Name)
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

	// Fetch store settings for shipping and tax calculation
	st, err := s.settingsSvc.GetSettings(ctx)
	if err != nil {
		// Fallback to defaults if settings fetch fails
		st = &domain.StoreSettings{
			FreeShippingThreshold: 1999,
			StandardDeliveryFee:   85,
			TaxPercentage:         5,
		}
	}

	shippingFee := st.StandardDeliveryFee
	if total >= st.FreeShippingThreshold {
		shippingFee = 0
	}

	tax := total * (st.TaxPercentage / 100)
	grandTotal := total + shippingFee + tax

	order := &domain.Order{
		ID:              1000000000 + rand.Int63n(9000000000),
		UserID:          userID,
		TotalPrice:      grandTotal,
		PaymentMethod:   paymentMethod,
		PaymentStatus:   "Pending",
		OrderStatus:     "Pending",
		ShippingAddress: shippingAddress,
		TrxID:           trxID,
		SenderNumber:    senderNumber,
		PaidAmount:      paidAmount,
	}

	createdOrder, err := s.orderRepo.Create(ctx, order, orderItems)
	if err != nil {
		return nil, err
	}

	// Clear cart after successful order
	s.cartRepo.Clear(ctx, userID)

	// Async email notification with timeout
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fullOrder, err := s.orderRepo.FindByID(ctx, createdOrder.ID)
		if err == nil && fullOrder != nil && fullOrder.User.Email != "" {
			s.mailer.SendOrderStatusUpdate(fullOrder, fullOrder.OrderStatus, "")
		}
	}()

	return createdOrder, nil
}

func (s *service) GetOrders(ctx context.Context, userID int64) ([]*domain.Order, error) {
	return s.orderRepo.ListByUser(ctx, userID)
}

func (s *service) GetOrderByID(ctx context.Context, orderID, userID int64) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, errors.New("unauthorized access to order")
	}
	return order, nil
}
func (s *service) AdminGetAllOrders(ctx context.Context) ([]*domain.Order, error) {
	return s.orderRepo.ListAll(ctx)
}

func (s *service) AdminConfirmOrder(ctx context.Context, orderID int64) error {
	err := s.orderRepo.UpdateStatus(ctx, orderID, "Confirmed", "Paid")
	if err == nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			time.Sleep(1 * time.Second)
			slog.Info("Order confirmed", "order_id", orderID)
			_ = ctx
		}()
	}
	return err
}

func (s *service) AdminUpdateOrderStatus(ctx context.Context, orderID int64, status string, estimatedDate string) error {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	statusOrder := map[string]int{
		"Pending":    1,
		"Confirmed":  2,
		"Processing": 3,
		"Shipped":    4,
		"Delivered":  5,
		"Cancelled":  6,
	}

	currentRank := statusOrder[order.OrderStatus]
	newRank := statusOrder[status]

	if newRank <= currentRank && status != "Cancelled" {
		return fmt.Errorf("cannot move status backwards from %s to %s", order.OrderStatus, status)
	}

	paymentStatus := order.PaymentStatus
	methodLower := strings.ToLower(order.PaymentMethod)

	if (status == "Confirmed" && (methodLower == "bkash" || methodLower == "nagad" || methodLower == "rocket")) || status == "Delivered" {
		paymentStatus = "Paid"
	}

	err = s.orderRepo.UpdateStatusWithStock(ctx, orderID, status, paymentStatus)
	if err != nil {
		return err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if order.User.Email != "" {
			s.mailer.SendOrderStatusUpdate(order, status, estimatedDate)
		}
		_ = ctx
	}()

	return nil
}

func (s *service) ConfirmPayment(ctx context.Context, orderID int64, trxID string, amount float64) error {
	err := s.orderRepo.UpdateStatus(ctx, orderID, "Pending", "Paid")
	if err == nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			time.Sleep(1 * time.Second)
			fullOrder, err := s.orderRepo.FindByID(ctx, orderID)
			if err == nil && fullOrder != nil && fullOrder.User.Email != "" {
				s.mailer.SendOrderStatusUpdate(fullOrder, "Pending", "")
			}
			slog.Info("Payment confirmed, order remains Pending for admin review", "order_id", orderID, "trx_id", trxID)
		}()
	}
	return err
}

func (s *service) AdminRequestDeleteOTP(ctx context.Context, adminID int64) error {
	return s.userSvc.RequestOTP(ctx, adminID, "order_deletion")
}

func (s *service) AdminDeleteOrder(ctx context.Context, id int64, otp string, adminID int64) error {
	// 1. Verify OTP
	valid, err := s.userSvc.VerifyOTP(ctx, adminID, "order_deletion", otp)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid or expired OTP for order deletion")
	}

	// 2. Get order to check status before deletion
	order, err := s.orderRepo.FindByID(ctx, id)
	if err == nil && order != nil {
		// If the order wasn't already cancelled/rejected, restore stock
		if order.OrderStatus != "Cancelled" && order.OrderStatus != "Rejected" {
			for _, item := range order.Items {
				s.productSvc.IncrementStock(ctx, item.ProductID, item.Quantity)
			}
		}
	}
	return s.orderRepo.Delete(ctx, id)
}
