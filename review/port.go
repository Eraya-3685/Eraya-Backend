package review

import "eraya/domain"

type Service interface {
	CreateReview(userID, productID int64, rating int, comment string) (*domain.Review, error)
	GetProductReviews(productID int64) ([]*domain.Review, error)
}

type ReviewRepo interface {
	Create(review *domain.Review) (*domain.Review, error)
	ListByProduct(productID int64) ([]*domain.Review, error)
}

// We need an interface to check if a user actually bought the item
type OrderVerifier interface {
	HasDeliveredOrder(userID, productID int64) (bool, error)
}
