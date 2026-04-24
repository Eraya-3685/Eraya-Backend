package product

import (
	"context"
	"eraya/domain"
)

type Service interface {
	CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error)
	GetProducts(ctx context.Context, page, limit int64, search string) ([]*domain.Product, int64, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	GetProductByID(ctx context.Context, id int64) (*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) ([]string, error)
	DeleteProduct(ctx context.Context, id int64) error

	// Categories
	CreateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error)
	ListCategories(ctx context.Context) ([]*domain.Category, error)
}

type ProductRepo interface {
	Create(ctx context.Context, product *domain.Product) (*domain.Product, error)
	List(ctx context.Context, page, limit int64, search string) ([]*domain.Product, error)
	Count(ctx context.Context, search string) (int64, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Product, error)
	FindByID(ctx context.Context, id int64) (*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) ([]string, error)
	Delete(ctx context.Context, id int64) error

	// Categories
	CreateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error)
	ListCategories(ctx context.Context) ([]*domain.Category, error)
}

type ProductCache interface {
	GetLatestProducts(ctx context.Context) ([]*domain.Product, error)
	SetLatestProducts(ctx context.Context, products []*domain.Product) error
}
