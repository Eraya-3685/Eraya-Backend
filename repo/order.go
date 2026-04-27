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

	var query string
	var args []any

	if o.ID > 0 {
		query = `
			INSERT INTO orders (id, user_id, total_price, payment_method, payment_status, order_status, shipping_address, trx_id, sender_number, paid_amount)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id, created_at
		`
		args = []any{o.ID, o.UserID, o.TotalPrice, o.PaymentMethod, o.PaymentStatus, o.OrderStatus, o.ShippingAddress, o.TrxID, o.SenderNumber, o.PaidAmount}
	} else {
		query = `
			INSERT INTO orders (user_id, total_price, payment_method, payment_status, order_status, shipping_address, trx_id, sender_number, paid_amount)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_at
		`
		args = []interface{}{o.UserID, o.TotalPrice, o.PaymentMethod, o.PaymentStatus, o.OrderStatus, o.ShippingAddress, o.TrxID, o.SenderNumber, o.PaidAmount}
	}

	err = tx.QueryRowContext(ctx, query, args...).Scan(&o.ID, &o.CreatedAt)
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
	if err != nil {
		return nil, err
	}

	for _, o := range orders {
		if err := r.populateOrderItems(ctx, o); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

func (r *orderRepo) ListAll(ctx context.Context) ([]*domain.Order, error) {
	query := `
		SELECT 
			o.id, o.user_id, o.total_price, o.payment_method, o.payment_status, 
			o.order_status, o.shipping_address, o.trx_id, o.sender_number, o.paid_amount, o.created_at, 
			o.tracking_number, o.confirmed_at, o.processing_at, o.shipped_at, o.delivered_at,
			u.full_name, u.email, u.phone
		FROM orders o
		JOIN users u ON o.user_id = u.id
		ORDER BY o.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		u := domain.User{}
		err := rows.Scan(
			&o.ID, &o.UserID, &o.TotalPrice, &o.PaymentMethod, &o.PaymentStatus,
			&o.OrderStatus, &o.ShippingAddress, &o.TrxID, &o.SenderNumber, &o.PaidAmount, &o.CreatedAt,
			&o.TrackingNumber, &o.ConfirmedAt, &o.ProcessingAt, &o.ShippedAt, &o.DeliveredAt,
			&u.FullName, &u.Email, &u.Phone,
		)
		if err != nil {
			fmt.Printf("Error scanning order: %v\n", err)
			continue
		}
		o.User = u
		orders = append(orders, o)
	}

	for _, o := range orders {
		if err := r.populateOrderItems(ctx, o); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

func (r *orderRepo) populateOrderItems(ctx context.Context, o *domain.Order) error {
	query := `
		SELECT 
			oi.id, oi.order_id, oi.product_id, oi.quantity, oi.price_at_purchase,
			p.name, p.slug, COALESCE(pi.image_url, '') as image_url
		FROM order_items oi
		JOIN products p ON oi.product_id = p.id
		LEFT JOIN product_images pi ON pi.product_id = p.id AND pi.is_primary = true
		WHERE oi.order_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, o.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		var prod domain.Product
		
		err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.PriceAtPurchase,
			&prod.Name, &prod.Slug, &prod.ImageUrl,
		)
		if err != nil {
			return err
		}
		item.Product = &prod
		o.Items = append(o.Items, item)
	}
	return nil
}

func (r *orderRepo) FindByID(ctx context.Context, id int64) (*domain.Order, error) {
	query := `
		SELECT 
			o.id, o.user_id, o.total_price, o.payment_method, o.payment_status, 
			o.order_status, o.shipping_address, o.trx_id, o.sender_number, o.paid_amount, o.created_at, 
			o.tracking_number, o.confirmed_at, o.processing_at, o.shipped_at, o.delivered_at,
			u.full_name, u.email, u.phone
		FROM orders o
		JOIN users u ON o.user_id = u.id
		WHERE o.id = $1
	`
	o := &domain.Order{}
	u := domain.User{}
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&o.ID, &o.UserID, &o.TotalPrice, &o.PaymentMethod, &o.PaymentStatus,
		&o.OrderStatus, &o.ShippingAddress, &o.TrxID, &o.SenderNumber, &o.PaidAmount, &o.CreatedAt,
		&o.TrackingNumber, &o.ConfirmedAt, &o.ProcessingAt, &o.ShippedAt, &o.DeliveredAt,
		&u.FullName, &u.Email, &u.Phone,
	)
	if err != nil {
		return nil, err
	}
	
	o.User = u
	
	if err := r.populateOrderItems(ctx, o); err != nil {
		return nil, err
	}
	
	return o, nil
}

func (r *orderRepo) UpdateStatus(ctx context.Context, id int64, status, paymentStatus string) error {
	var query string
	if status == "Confirmed" {
		query = `UPDATE orders SET order_status = $1, payment_status = $2, confirmed_at = NOW() WHERE id = $3`
	} else if status == "Processing" {
		query = `UPDATE orders SET order_status = $1, payment_status = $2, processing_at = NOW() WHERE id = $3`
	} else if status == "Shipped" {
		query = `UPDATE orders SET order_status = $1, payment_status = $2, shipped_at = NOW() WHERE id = $3`
	} else if status == "Delivered" {
		query = `UPDATE orders SET order_status = $1, payment_status = $2, delivered_at = NOW() WHERE id = $3`
	} else {
		query = `UPDATE orders SET order_status = $1, payment_status = $2 WHERE id = $3`
	}
	_, err := r.db.ExecContext(ctx, query, status, paymentStatus, id)
	return err
}

func (r *orderRepo) UpdateStatusWithStock(ctx context.Context, id int64, status, paymentStatus string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	// 1. Get current order status and items
	var oldStatus string
	err = tx.GetContext(ctx, &oldStatus, "SELECT order_status FROM orders WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 2. Update status
	var statusQuery string
	if status == "Confirmed" {
		statusQuery = `UPDATE orders SET order_status = $1, payment_status = $2, confirmed_at = NOW() WHERE id = $3`
	} else if status == "Processing" {
		statusQuery = `UPDATE orders SET order_status = $1, payment_status = $2, processing_at = NOW() WHERE id = $3`
	} else if status == "Shipped" {
		statusQuery = `UPDATE orders SET order_status = $1, payment_status = $2, shipped_at = NOW() WHERE id = $3`
	} else if status == "Delivered" {
		statusQuery = `UPDATE orders SET order_status = $1, payment_status = $2, delivered_at = NOW() WHERE id = $3`
	} else {
		statusQuery = `UPDATE orders SET order_status = $1, payment_status = $2 WHERE id = $3`
	}

	_, err = tx.ExecContext(ctx, statusQuery, status, paymentStatus, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 3. Handle stock logic
	// If moving to Cancelled/Rejected from a non-cancelled state, restore stock
	if (status == "Cancelled" || status == "Rejected") && (oldStatus != "Cancelled" && oldStatus != "Rejected") {
		// Fetch items to restore stock
		var items []domain.OrderItem
		err = tx.SelectContext(ctx, &items, "SELECT product_id, quantity FROM order_items WHERE order_id = $1", id)
		if err == nil {
			for _, item := range items {
				_, err = tx.ExecContext(ctx, "UPDATE products SET stock_count = stock_count + $1 WHERE id = $2", item.Quantity, item.ProductID)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	// If moving AWAY from Cancelled/Rejected to an active state, decrement stock
	if (oldStatus == "Cancelled" || oldStatus == "Rejected") && (status != "Cancelled" && status != "Rejected") {
		// Fetch items to decrement stock
		var items []domain.OrderItem
		err = tx.SelectContext(ctx, &items, "SELECT product_id, quantity FROM order_items WHERE order_id = $1", id)
		if err == nil {
			for _, item := range items {
				// Check stock first or rely on the constraint in SQL
				res, err := tx.ExecContext(ctx, "UPDATE products SET stock_count = stock_count - $1 WHERE id = $2 AND stock_count >= $1", item.Quantity, item.ProductID)
				if err != nil {
					tx.Rollback()
					return err
				}
				rows, _ := res.RowsAffected()
				if rows == 0 {
					tx.Rollback()
					return fmt.Errorf("insufficient stock for product ID: %d", item.ProductID)
				}
			}
		}
	}

	return tx.Commit()
}

func (r *orderRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM orders WHERE id = $1", id)
	return err
}
