package coupon

import (
	"context"
	"errors"
	"eraya/domain"
	"fmt"
	"strings"
	"time"
)

type service struct {
	repo domain.CouponRepo
}

func NewService(repo domain.CouponRepo) domain.CouponService {
	return &service{repo: repo}
}

func (s *service) CreateCoupon(ctx context.Context, code, discountType string, discountValue, minCartValue float64, expiresAt time.Time) (*domain.Coupon, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, errors.New("coupon code is required")
	}
	discountType = strings.ToLower(strings.TrimSpace(discountType))
	if discountType != "percentage" && discountType != "flat" {
		return nil, errors.New("discount type must be percentage or flat")
	}
	if discountValue <= 0 {
		return nil, errors.New("discount value must be positive")
	}
	if expiresAt.Before(time.Now()) {
		return nil, errors.New("expiry time must be in the future")
	}

	c := &domain.Coupon{
		Code:          code,
		DiscountType:  discountType,
		DiscountValue: discountValue,
		MinCartValue:  minCartValue,
		IsActive:      true,
		ExpiresAt:     expiresAt,
	}
	return s.repo.Create(ctx, c)
}

func (s *service) GetCouponByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	return s.repo.FindByCode(ctx, code)
}

func (s *service) ListCoupons(ctx context.Context, page, limit int64, search string) ([]*domain.Coupon, int64, error) {
	return s.repo.List(ctx, page, limit, search)
}

func (s *service) DeleteCoupon(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) ValidateAndApplyCoupon(ctx context.Context, code string, cartTotal float64) (*domain.Coupon, float64, error) {
	c, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, 0, errors.New("invalid coupon code")
	}

	if !c.IsActive {
		return nil, 0, errors.New("coupon is inactive")
	}

	if time.Now().After(c.ExpiresAt) {
		return nil, 0, errors.New("coupon has expired")
	}

	if cartTotal < c.MinCartValue {
		return nil, 0, fmt.Errorf("minimum purchase of ৳%.2f is required for this coupon", c.MinCartValue)
	}

	var discountAmount float64
	if c.DiscountType == "percentage" {
		discountAmount = cartTotal * (c.DiscountValue / 100.0)
	} else {
		discountAmount = c.DiscountValue
	}

	if discountAmount > cartTotal {
		discountAmount = cartTotal
	}

	return c, discountAmount, nil
}
