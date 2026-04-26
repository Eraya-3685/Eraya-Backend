package repo

import (
	"context"
	"eraya/domain"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type orderRepo struct {
	db *sqlx.DB
}

func NewOrderRepo(db *sqlx.DB) domain.OrderRepo {
	return &orderRepo{db: db}
}

func (r *orderRepo) Create(ctx context.Context, o *domain.Order, items []*domain.OrderItem) (*domain.Order, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO orders (user_id, total_price, payment_method, payment_status, order_status, shipping_address)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err = tx.QueryRowContext(ctx, query, o.UserID, o.TotalPrice, o.PaymentMethod, o.PaymentStatus, o.OrderStatus, o.ShippingAddress).
		Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	itemQuery := `
		INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase)
		VALUES ($1, $2, $3, $4)
	`
	stockUpdateQuery := `
		UPDATE products 
		SET stock_count = stock_count - $1 
		WHERE id = $2 AND stock_count >= $1
	`

	for _, item := range items {
		item.OrderID = o.ID
		// 1. Insert order item
		_, err = tx.ExecContext(ctx, itemQuery, o.ID, item.ProductID, item.Quantity, item.PriceAtPurchase)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		// 2. Decrement stock
		res, err := tx.ExecContext(ctx, stockUpdateQuery, item.Quantity, item.ProductID)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			tx.Rollback()
			return nil, fmt.Errorf("insufficient stock for product ID: %d", item.ProductID)
		}
	}

	err = tx.Commit()
	return o, err
}

func (r *orderRepo) ListByUser(ctx context.Context, userID int64) ([]*domain.Order, error) {
	query := `SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC`
	var orders []*domain.Order
	err := r.db.SelectContext(ctx, &orders, query, userID)
	return orders, err
}

func (r *orderRepo) ListAll(ctx context.Context) ([]*domain.Order, error) {
	query := `
		SELECT o.*, u.full_name as "user.full_name", u.email as "user.email"
		FROM orders o
		JOIN users u ON o.user_id = u.id
		ORDER BY o.created_at DESC
	`
	var orders []*domain.Order
	err := r.db.SelectContext(ctx, &orders, query)
	return orders, err
}

func (r *orderRepo) FindByID(ctx context.Context, id int64) (*domain.Order, error) {
	query := `SELECT * FROM orders WHERE id = $1`
	var o domain.Order
	err := r.db.GetContext(ctx, &o, query, id)
	return &o, err
}

func (r *orderRepo) UpdateStatus(ctx context.Context, id int64, status, paymentStatus string) error {
	query := `UPDATE orders SET order_status = $1, payment_status = $2, confirmed_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, paymentStatus, id)
	return err
}

func (r *orderRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM orders WHERE id = $1", id)
	return err
}
