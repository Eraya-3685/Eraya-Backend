package repo

import (
	"context"
	"eraya/domain"
	"eraya/wishlist"

	"github.com/jmoiron/sqlx"
)

type wishlistRepo struct {
	db *sqlx.DB
}

func NewWishlistRepo(db *sqlx.DB) wishlist.WishlistRepo {
	return &wishlistRepo{db: db}
}

func (r *wishlistRepo) Add(ctx context.Context, userID, productID int64) error {
	query := `INSERT INTO wishlists (user_id, product_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, userID, productID)
	return err
}

func (r *wishlistRepo) Remove(ctx context.Context, userID, productID int64) error {
	query := `DELETE FROM wishlists WHERE user_id = $1 AND product_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, productID)
	return err
}

func (r *wishlistRepo) List(ctx context.Context, userID int64) ([]*domain.Product, error) {
	query := `
		SELECT p.* 
		FROM products p
		JOIN wishlists w ON p.id = w.product_id
		WHERE w.user_id = $1
		ORDER BY w.created_at DESC
	`
	var products []*domain.Product
	err := r.db.SelectContext(ctx, &products, query, userID)
	if err != nil {
		return nil, err
	}

	// Fetch images for each product
	for _, p := range products {
		var images []domain.ProductImage
		err := r.db.SelectContext(ctx, &images, "SELECT * FROM product_images WHERE product_id = $1", p.ID)
		if err == nil {
			p.Images = images
		}
	}

	return products, nil
}

func (r *wishlistRepo) IsWishlisted(ctx context.Context, userID, productID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM wishlists WHERE user_id = $1 AND product_id = $2)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, userID, productID)
	return exists, err
}

func (r *wishlistRepo) Clear(ctx context.Context, userID int64) error {
	query := `DELETE FROM wishlists WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}
