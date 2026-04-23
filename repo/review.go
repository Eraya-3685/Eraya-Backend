package repo

import (
	"context"
	"eraya/domain"
	"eraya/review"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type reviewRepo struct {
	db *sqlx.DB
}

func NewReviewRepo(db *sqlx.DB) review.ReviewRepo {
	return &reviewRepo{db: db}
}

func (r *reviewRepo) Create(ctx context.Context, rev *domain.Review) (*domain.Review, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO reviews (product_id, user_id, rating, comment)
		VALUES (:product_id, :user_id, :rating, :comment)
		RETURNING id, created_at
	`
	rows, err := sqlx.NamedQueryContext(ctx, tx, query, rev)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create review: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		rows.Scan(&rev.ID, &rev.CreatedAt)
	}

	updateProductQuery := `
		UPDATE products 
		SET total_reviews = total_reviews + 1, 
			average_rating = ((average_rating * total_reviews) + $1) / (total_reviews + 1)
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, updateProductQuery, rev.Rating, rev.ProductID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	return rev, err
}

func (r *reviewRepo) ListByProduct(ctx context.Context, productID int64) ([]*domain.Review, error) {
	query := `SELECT * FROM reviews WHERE product_id = $1 ORDER BY created_at DESC`
	var reviews []*domain.Review
	err := r.db.SelectContext(ctx, &reviews, query, productID)
	return reviews, err
}

func (r *reviewRepo) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	// Fetch review first to get rating and product ID
	var rev domain.Review
	err = tx.GetContext(ctx, &rev, "SELECT * FROM reviews WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete review
	_, err = tx.ExecContext(ctx, "DELETE FROM reviews WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Update product rating stats
	updateProductQuery := `
		UPDATE products 
		SET total_reviews = total_reviews - 1, 
			average_rating = CASE 
				WHEN total_reviews - 1 > 0 
				THEN ((average_rating * total_reviews) - $1) / (total_reviews - 1)
				ELSE 0 
			END
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, updateProductQuery, rev.Rating, rev.ProductID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// OrderVerifier Implementation
type orderVerifier struct {
	db *sqlx.DB
}

func NewOrderVerifier(db *sqlx.DB) review.OrderVerifier {
	return &orderVerifier{db: db}
}

func (v *orderVerifier) HasDeliveredOrder(ctx context.Context, userID, productID int64) (bool, error) {
	query := `
		SELECT COUNT(o.id) 
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		WHERE o.user_id = $1 AND oi.product_id = $2 AND o.order_status = 'delivered'
	`
	var count int
	err := v.db.GetContext(ctx, &count, query, userID, productID)
	return count > 0, err
}
