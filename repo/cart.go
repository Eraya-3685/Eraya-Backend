package repo

import (
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

func (r *cartRepo) Add(item *domain.CartItem) error {
	// Simple add, normally we'd check if it exists and increment quantity
	query := `INSERT INTO cart_items (user_id, product_id, quantity) VALUES (:user_id, :product_id, :quantity)`
	_, err := r.db.NamedExec(query, &item)
	return err
}

func (r *cartRepo) List(userID int64) ([]*domain.CartItem, error) {
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
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *cartRepo) Clear(userID int64) error {
	query := `DELETE FROM cart_items WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}
