package review

import (
	"context"
	"errors"
	"eraya/domain"
)

type service struct {
	repo     ReviewRepo
	verifier OrderVerifier
}

func NewService(repo ReviewRepo, verifier OrderVerifier) Service {
	return &service{
		repo:     repo,
		verifier: verifier,
	}
}

func (s *service) CreateReview(ctx context.Context, userID, productID int64, rating int, comment string) (*domain.Review, error) {
	if rating < 1 || rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	// Verify purchase
	hasBought, err := s.verifier.HasDeliveredOrder(ctx, userID, productID)
	if err != nil {
		return nil, err
	}
	if !hasBought {
		return nil, errors.New("you can only review products you have bought and received")
	}

	r := &domain.Review{
		UserID:     userID,
		ProductID:  productID,
		Rating:     rating,
		Comment:    &comment,
		IsVerified: true,
		IsApproved: true,
	}

	return s.repo.Create(ctx, r)
}

func (s *service) GetProductReviews(ctx context.Context, productID int64) ([]*domain.Review, error) {
	return s.repo.ListByProduct(ctx, productID)
}

func (s *service) DeleteReview(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
