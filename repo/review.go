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
	query := `
		INSERT INTO reviews (product_id, user_id, rating, comment, is_verified, is_approved, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query, rev.ProductID, rev.UserID, rev.Rating, rev.Comment, rev.IsVerified, rev.IsApproved, rev.ImageURL).
		Scan(&rev.ID, &rev.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	return rev, nil
}

func (r *reviewRepo) ListByProduct(ctx context.Context, productID int64) ([]*domain.Review, error) {
	query := `
		SELECT r.id, r.user_id, r.product_id, r.rating, r.comment, r.created_at, r.is_verified, r.is_approved, r.image_url,
		       u.id, u.full_name
		FROM reviews r
		JOIN users u ON r.user_id = u.id
		WHERE r.product_id = $1 AND r.is_approved = true
		ORDER BY r.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*domain.Review
	for rows.Next() {
		rev := &domain.Review{}
		u := &domain.User{}
		err := rows.Scan(
			&rev.ID, &rev.UserID, &rev.ProductID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.IsVerified, &rev.IsApproved, &rev.ImageURL,
			&u.ID, &u.FullName,
		)
		if err != nil {
			return nil, err
		}
		rev.User = u
		reviews = append(reviews, rev)
	}
	return reviews, nil
}

func (r *reviewRepo) ListAll(ctx context.Context, page, limit int64, search, filter string) ([]*domain.Review, int64, error) {
	var whereClauses []string
	var args []any

	if filter == "Pending" {
		whereClauses = append(whereClauses, "r.is_approved = false")
	} else if filter == "Approved" {
		whereClauses = append(whereClauses, "r.is_approved = true")
	}

	if search != "" {
		param := "%" + search + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(u.full_name ILIKE $%d OR r.comment ILIKE $%d)", len(args)+1, len(args)+2))
		args = append(args, param, param)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			whereSQL += " AND " + whereClauses[i]
		}
	}

	countQuery := "SELECT COUNT(*) FROM reviews r JOIN users u ON r.user_id = u.id" + whereSQL
	var count int64
	err := r.db.GetContext(ctx, &count, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT r.id, r.user_id, r.product_id, r.rating, r.comment, r.created_at, r.is_verified, r.is_approved, r.image_url,
		       u.id, u.full_name
		FROM reviews r
		JOIN users u ON r.user_id = u.id
	` + whereSQL + " ORDER BY r.created_at DESC"

	if limit > 0 {
		offset := (page - 1) * limit
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reviews []*domain.Review
	for rows.Next() {
		rev := &domain.Review{}
		u := &domain.User{}
		err := rows.Scan(
			&rev.ID, &rev.UserID, &rev.ProductID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.IsVerified, &rev.IsApproved, &rev.ImageURL,
			&u.ID, &u.FullName,
		)
		if err != nil {
			return nil, 0, err
		}
		rev.User = u
		reviews = append(reviews, rev)
	}
	return reviews, count, nil
}

func (r *reviewRepo) Approve(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	// 1. Check if already approved
	var rev domain.Review
	err = tx.GetContext(ctx, &rev, "SELECT * FROM reviews WHERE id = $1 FOR UPDATE", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	if rev.IsApproved {
		tx.Rollback()
		return nil // Already approved, nothing to do
	}

	// 2. Mark as approved
	_, err = tx.ExecContext(ctx, "UPDATE reviews SET is_approved = true WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 3. Update product rating stats
	updateProductQuery := `
		UPDATE products 
		SET total_reviews = total_reviews + 1, 
			average_rating = ((average_rating * total_reviews) + $1) / (total_reviews + 1)
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, updateProductQuery, rev.Rating, rev.ProductID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *reviewRepo) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	// Fetch review first
	var rev domain.Review
	err = tx.GetContext(ctx, &rev, "SELECT * FROM reviews WHERE id = $1 FOR UPDATE", id)
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

	// Update product rating stats ONLY IF it was approved
	if rev.IsApproved {
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
		WHERE o.user_id = $1 AND oi.product_id = $2 AND LOWER(o.order_status) = 'delivered'
	`
	var count int
	err := v.db.GetContext(ctx, &count, query, userID, productID)
	return count > 0, err
}
