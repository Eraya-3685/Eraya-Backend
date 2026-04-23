package repo

import (
	"context"
	"eraya/domain"
	"eraya/order"

	"github.com/jmoiron/sqlx"
)

type orderRepo struct {
	db *sqlx.DB
}

func NewOrderRepo(db *sqlx.DB) order.OrderRepo {
	return &orderRepo{db: db}
}

func (r *orderRepo) Create(ctx context.Context, o *domain.Order, items []*domain.OrderItem) (*domain.Order, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO orders (user_id, total_price, payment_method, payment_status, order_status, shipping_address)
		VALUES (:user_id, :total_price, :payment_method, :payment_status, :order_status, :shipping_address)
		RETURNING id, created_at
	`
	rows, err := sqlx.NamedQueryContext(ctx, tx, query, o)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if rows.Next() {
		rows.Scan(&o.ID, &o.CreatedAt)
	}
	rows.Close()

	itemQuery := `
		INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase)
		VALUES (:order_id, :product_id, :quantity, :price_at_purchase)
	`
	for _, item := range items {
		item.OrderID = o.ID
		_, err = tx.NamedExecContext(ctx, itemQuery, item)
		if err != nil {
			tx.Rollback()
			return nil, err
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
	query := `SELECT * FROM orders ORDER BY created_at DESC`
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
