package review

import (
	"context"
	"eraya/domain"
)

type Service interface {
	CreateReview(ctx context.Context, userID, productID int64, rating int, comment string, imageURL *string) (*domain.Review, error)
	GetProductReviews(ctx context.Context, productID int64) ([]*domain.Review, error)
	ListAllReviews(ctx context.Context, page, limit int64, search, filter string) ([]*domain.Review, int64, error)
	ApproveReview(ctx context.Context, id int64) error
	DeleteReview(ctx context.Context, id int64) error
}

type ReviewRepo interface {
	Create(ctx context.Context, review *domain.Review) (*domain.Review, error)
	ListByProduct(ctx context.Context, productID int64) ([]*domain.Review, error)
	ListAll(ctx context.Context, page, limit int64, search, filter string) ([]*domain.Review, int64, error)
	Approve(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
}

// We need an interface to check if a user actually bought the item
type OrderVerifier interface {
	HasDeliveredOrder(ctx context.Context, userID, productID int64) (bool, error)
}
