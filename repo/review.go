package repo

import (
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

func (r *reviewRepo) Create(rev *domain.Review) (*domain.Review, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO reviews (product_id, user_id, rating, comment)
		VALUES (:product_id, :user_id, :rating, :comment)
		RETURNING id, created_at
	`
	rows, err := tx.NamedQuery(query, rev)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create review: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		rows.Scan(&rev.ID, &rev.CreatedAt)
	}

	// Trigger to update product average_rating and total_reviews could be here or DB trigger
	updateProductQuery := `
		UPDATE products 
		SET total_reviews = total_reviews + 1, 
			average_rating = ((average_rating * total_reviews) + $1) / (total_reviews + 1)
		WHERE id = $2
	`
	_, err = tx.Exec(updateProductQuery, rev.Rating, rev.ProductID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	return rev, err
}

func (r *reviewRepo) ListByProduct(productID int64) ([]*domain.Review, error) {
	query := `SELECT * FROM reviews WHERE product_id = $1 ORDER BY created_at DESC`
	var reviews []*domain.Review
	err := r.db.Select(&reviews, query, productID)
	return reviews, err
}

// OrderVerifier Implementation
type orderVerifier struct {
	db *sqlx.DB
}

func NewOrderVerifier(db *sqlx.DB) review.OrderVerifier {
	return &orderVerifier{db: db}
}

func (v *orderVerifier) HasDeliveredOrder(userID, productID int64) (bool, error) {
	query := `
		SELECT COUNT(o.id) 
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		WHERE o.user_id = $1 AND oi.product_id = $2 AND o.order_status = 'delivered'
	`
	var count int
	err := v.db.Get(&count, query, userID, productID)
	return count > 0, err
}
