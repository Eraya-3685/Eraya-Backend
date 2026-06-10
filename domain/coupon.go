package domain

import (
	"context"
	"time"
)

type Coupon struct {
	ID            int64     `json:"id" db:"id"`
	Code          string    `json:"code" db:"code"`
	DiscountType  string    `json:"discount_type" db:"discount_type"` // "percentage" or "flat"
	DiscountValue float64   `json:"discount_value" db:"discount_value"`
	MinCartValue  float64   `json:"min_cart_value" db:"min_cart_value"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	ExpiresAt     time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type CouponRepo interface {
	Create(ctx context.Context, c *Coupon) (*Coupon, error)
	FindByCode(ctx context.Context, code string) (*Coupon, error)
	FindByID(ctx context.Context, id int64) (*Coupon, error)
	List(ctx context.Context, page, limit int64, search string) ([]*Coupon, int64, error)
	Delete(ctx context.Context, id int64) error
}

type CouponService interface {
	CreateCoupon(ctx context.Context, code, discountType string, discountValue, minCartValue float64, expiresAt time.Time) (*Coupon, error)
	GetCouponByCode(ctx context.Context, code string) (*Coupon, error)
	ListCoupons(ctx context.Context, page, limit int64, search string) ([]*Coupon, int64, error)
	DeleteCoupon(ctx context.Context, id int64) error
	ValidateAndApplyCoupon(ctx context.Context, code string, cartTotal float64) (*Coupon, float64, error)
}
