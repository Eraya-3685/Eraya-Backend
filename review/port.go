package review

import (
	"context"
	"eraya/domain"
)

type Service interface {
	CreateReview(ctx context.Context, userID, productID int64, rating int, comment string) (*domain.Review, error)
	GetProductReviews(ctx context.Context, productID int64) ([]*domain.Review, error)
	DeleteReview(ctx context.Context, id int64) error
}

type ReviewRepo interface {
	Create(ctx context.Context, review *domain.Review) (*domain.Review, error)
	ListByProduct(ctx context.Context, productID int64) ([]*domain.Review, error)
	Delete(ctx context.Context, id int64) error
}

// We need an interface to check if a user actually bought the item
type OrderVerifier interface {
	HasDeliveredOrder(ctx context.Context, userID, productID int64) (bool, error)
}
