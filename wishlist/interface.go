package wishlist

import (
	"context"
	"eraya/domain"
)

type WishlistRepo interface {
	Add(ctx context.Context, userID, productID int64) error
	Remove(ctx context.Context, userID, productID int64) error
	List(ctx context.Context, userID int64) ([]*domain.Product, error)
	IsWishlisted(ctx context.Context, userID, productID int64) (bool, error)
	Clear(ctx context.Context, userID int64) error
}

type Service interface {
	AddToWishlist(ctx context.Context, userID, productID int64) error
	RemoveFromWishlist(ctx context.Context, userID, productID int64) error
	GetUserWishlist(ctx context.Context, userID int64) ([]*domain.Product, error)
	ClearWishlist(ctx context.Context, userID int64) error
}
