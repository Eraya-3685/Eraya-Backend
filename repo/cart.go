package repo

import (
	"context"
	"eraya/domain"

	"github.com/jmoiron/sqlx"
)

type cartRepo struct {
	db *sqlx.DB
}

func NewCartRepo(db *sqlx.DB) domain.CartRepo {
	return &cartRepo{db: db}
}

func (r *cartRepo) Add(ctx context.Context, item *domain.CartItem) error {
	query := `
		INSERT INTO cart_items (user_id, product_id, quantity, selected_color, selected_size) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, product_id, selected_color, selected_size) 
		DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
	`
	_, err := r.db.ExecContext(ctx, query, item.UserID, item.ProductID, item.Quantity, item.SelectedColor, item.SelectedSize)
	return err
}

func (r *cartRepo) List(ctx context.Context, userID int64) ([]*domain.CartItem, error) {
	query := `
		SELECT 
			c.id, c.user_id, c.product_id, c.quantity, c.selected_color, c.selected_size,
			p.id AS "product.id",
			p.name AS "product.name",
			p.base_price AS "product.base_price",
			p.slug AS "product.slug"
		FROM cart_items c
		JOIN products p ON c.product_id = p.id
		WHERE c.user_id = $1
	`
	var items []*domain.CartItem
	err := r.db.SelectContext(ctx, &items, query, userID)
	return items, err
}

func (r *cartRepo) Clear(ctx context.Context, userID int64) error {
	query := `DELETE FROM cart_items WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}
