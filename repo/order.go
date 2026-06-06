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
			INSERT INTO orders (id, user_id, total_price, payment_method, payment_status, order_status, shipping_address, trx_id, sender_number, paid_amount, coupon_code, discount_amount)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id, created_at
		`
		args = []any{o.ID, o.UserID, o.TotalPrice, o.PaymentMethod, o.PaymentStatus, o.OrderStatus, o.ShippingAddress, o.TrxID, o.SenderNumber, o.PaidAmount, o.CouponCode, o.DiscountAmount}
	} else {
		query = `
			INSERT INTO orders (user_id, total_price, payment_method, payment_status, order_status, shipping_address, trx_id, sender_number, paid_amount, coupon_code, discount_amount)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id, created_at
		`
		args = []interface{}{o.UserID, o.TotalPrice, o.PaymentMethod, o.PaymentStatus, o.OrderStatus, o.ShippingAddress, o.TrxID, o.SenderNumber, o.PaidAmount, o.CouponCode, o.DiscountAmount}
	}

	err = tx.QueryRowContext(ctx, query, args...).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	itemQuery := `
		INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase, selected_color, selected_size)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, item := range items {
		item.OrderID = o.ID
		// 1. Insert order item
		_, err = tx.ExecContext(ctx, itemQuery, o.ID, item.ProductID, item.Quantity, item.PriceAtPurchase, item.SelectedColor, item.SelectedSize)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		// Load product first to lock it and get its current variation stock
		var prod domain.Product
		err = tx.GetContext(ctx, &prod, "SELECT id, name, stock_count, variation_stock FROM products WHERE id = $1 FOR UPDATE", item.ProductID)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		if len(prod.VariationStock) > 0 {
			found := false
			for idx, v := range prod.VariationStock {
				if v.Color == item.SelectedColor && v.Size == item.SelectedSize {
					found = true
					if v.Stock < item.Quantity {
						tx.Rollback()
						return nil, fmt.Errorf("insufficient stock for product %s variation (%s, %s)", prod.Name, item.SelectedColor, item.SelectedSize)
					}
					prod.VariationStock[idx].Stock -= item.Quantity
					break
				}
			}
			if !found && (item.SelectedColor != "" || item.SelectedSize != "") {
				tx.Rollback()
				return nil, fmt.Errorf("invalid variation combination (%s, %s) for product %s", item.SelectedColor, item.SelectedSize, prod.Name)
			}
		}

		if prod.StockCount < item.Quantity {
			tx.Rollback()
			return nil, fmt.Errorf("insufficient stock for product: %s", prod.Name)
		}

		// 2. Decrement stock
		_, err = tx.ExecContext(ctx, "UPDATE products SET stock_count = stock_count - $1, variation_stock = $2::jsonb WHERE id = $3", item.Quantity, prod.VariationStock, item.ProductID)
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
			oi.selected_color, oi.selected_size,
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
			&item.SelectedColor, &item.SelectedSize,
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
		err = tx.SelectContext(ctx, &items, "SELECT product_id, quantity, selected_color, selected_size FROM order_items WHERE order_id = $1", id)
		if err == nil {
			for _, item := range items {
				var prod domain.Product
				err = tx.GetContext(ctx, &prod, "SELECT id, name, stock_count, variation_stock FROM products WHERE id = $1 FOR UPDATE", item.ProductID)
				if err != nil {
					tx.Rollback()
					return err
				}

				if len(prod.VariationStock) > 0 {
					for idx, v := range prod.VariationStock {
						if v.Color == item.SelectedColor && v.Size == item.SelectedSize {
							prod.VariationStock[idx].Stock += item.Quantity
							break
						}
					}
				}

				_, err = tx.ExecContext(ctx, "UPDATE products SET stock_count = stock_count + $1, variation_stock = $2::jsonb WHERE id = $3", item.Quantity, prod.VariationStock, item.ProductID)
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
		err = tx.SelectContext(ctx, &items, "SELECT product_id, quantity, selected_color, selected_size FROM order_items WHERE order_id = $1", id)
		if err == nil {
			for _, item := range items {
				var prod domain.Product
				err = tx.GetContext(ctx, &prod, "SELECT id, name, stock_count, variation_stock FROM products WHERE id = $1 FOR UPDATE", item.ProductID)
				if err != nil {
					tx.Rollback()
					return err
				}

				if len(prod.VariationStock) > 0 {
					found := false
					for idx, v := range prod.VariationStock {
						if v.Color == item.SelectedColor && v.Size == item.SelectedSize {
							found = true
							if v.Stock < item.Quantity {
								tx.Rollback()
								return fmt.Errorf("insufficient stock for product %s variation (%s, %s)", prod.Name, item.SelectedColor, item.SelectedSize)
							}
							prod.VariationStock[idx].Stock -= item.Quantity
							break
						}
					}
					if !found && (item.SelectedColor != "" || item.SelectedSize != "") {
						tx.Rollback()
						return fmt.Errorf("invalid variation combination (%s, %s) for product %s", item.SelectedColor, item.SelectedSize, prod.Name)
					}
				}

				if prod.StockCount < item.Quantity {
					tx.Rollback()
					return fmt.Errorf("insufficient stock for product: %s", prod.Name)
				}

				_, err = tx.ExecContext(ctx, "UPDATE products SET stock_count = stock_count - $1, variation_stock = $2::jsonb WHERE id = $3", item.Quantity, prod.VariationStock, item.ProductID)
				if err != nil {
					tx.Rollback()
					return err
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

func (r *orderRepo) GetCategorySales(ctx context.Context) ([]domain.CategorySales, error) {
	query := `
		SELECT c.name as category_name, COALESCE(SUM(oi.quantity * oi.price_at_purchase), 0) as total_sales, COUNT(DISTINCT oi.product_id) as product_count
		FROM order_items oi
		JOIN product_categories pc ON oi.product_id = pc.product_id
		JOIN categories c ON pc.category_id = c.id
		JOIN orders o ON oi.order_id = o.id
		WHERE o.payment_status = 'Paid'
		GROUP BY c.name
		ORDER BY total_sales DESC
	`
	var sales []domain.CategorySales
	err := r.db.SelectContext(ctx, &sales, query)
	return sales, err
}

func (r *orderRepo) GetLowStockProducts(ctx context.Context) ([]domain.Product, error) {
	query := `SELECT * FROM products WHERE stock_count < 5 AND is_active = true ORDER BY stock_count ASC`
	var products []domain.Product
	err := r.db.SelectContext(ctx, &products, query)
	return products, err
}
