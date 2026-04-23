package review

import (
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

func (s *service) CreateReview(userID, productID int64, rating int, comment string) (*domain.Review, error) {
	if rating < 1 || rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	// Verify purchase
	hasBought, err := s.verifier.HasDeliveredOrder(userID, productID)
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
		IsApproved: true, // Default to true, admin can hide later
	}

	return s.repo.Create(r)
}

func (s *service) GetProductReviews(productID int64) ([]*domain.Review, error) {
	return s.repo.ListByProduct(productID)
}
