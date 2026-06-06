package repo

import (
	"context"
	"database/sql"
	"eraya/domain"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type couponRepo struct {
	db *sqlx.DB
}

func NewCouponRepo(db *sqlx.DB) domain.CouponRepo {
	return &couponRepo{db: db}
}

func (r *couponRepo) Create(ctx context.Context, c *domain.Coupon) (*domain.Coupon, error) {
	query := `
		INSERT INTO coupons (code, discount_type, discount_value, min_cart_value, is_active, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query, c.Code, c.DiscountType, c.DiscountValue, c.MinCartValue, c.IsActive, c.ExpiresAt).
		Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create coupon: %w", err)
	}
	return c, nil
}

func (r *couponRepo) FindByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	query := `SELECT * FROM coupons WHERE UPPER(code) = UPPER($1)`
	var c domain.Coupon
	err := r.db.GetContext(ctx, &c, query, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("coupon not found")
		}
		return nil, err
	}
	return &c, nil
}

func (r *couponRepo) FindByID(ctx context.Context, id int64) (*domain.Coupon, error) {
	query := `SELECT * FROM coupons WHERE id = $1`
	var c domain.Coupon
	err := r.db.GetContext(ctx, &c, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("coupon not found")
		}
		return nil, err
	}
	return &c, nil
}

func (r *couponRepo) List(ctx context.Context) ([]*domain.Coupon, error) {
	query := `SELECT * FROM coupons ORDER BY created_at DESC`
	var coupons []*domain.Coupon
	err := r.db.SelectContext(ctx, &coupons, query)
	if err != nil {
		return nil, err
	}
	return coupons, nil
}

func (r *couponRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM coupons WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
