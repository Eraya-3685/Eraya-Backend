package repo

import (
	"context"
	"eraya/domain"
	"eraya/order"

	"github.com/jmoiron/sqlx"
)

type cartRepo struct {
	db *sqlx.DB
}

func NewCartRepo(db *sqlx.DB) order.CartRepo {
	return &cartRepo{db: db}
}

func (r *cartRepo) Add(ctx context.Context, item *domain.CartItem) error {
	query := `INSERT INTO cart_items (user_id, product_id, quantity) VALUES (:user_id, :product_id, :quantity)`
	_, err := r.db.NamedExecContext(ctx, query, &item)
	return err
}

func (r *cartRepo) List(ctx context.Context, userID int64) ([]*domain.CartItem, error) {
	query := `
		SELECT 
			c.id, c.user_id, c.product_id, c.quantity,
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
