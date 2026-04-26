package wishlist

import (
	"context"
	"eraya/domain"
)

type service struct {
	repo WishlistRepo
}

func NewService(repo WishlistRepo) Service {
	return &service{repo: repo}
}

func (s *service) AddToWishlist(ctx context.Context, userID, productID int64) error {
	return s.repo.Add(ctx, userID, productID)
}

func (s *service) RemoveFromWishlist(ctx context.Context, userID, productID int64) error {
	return s.repo.Remove(ctx, userID, productID)
}

func (s *service) GetUserWishlist(ctx context.Context, userID int64) ([]*domain.Product, error) {
	return s.repo.List(ctx, userID)
}

func (s *service) ClearWishlist(ctx context.Context, userID int64) error {
	return s.repo.Clear(ctx, userID)
}
