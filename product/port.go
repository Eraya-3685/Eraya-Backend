package product

import (
	"context"
	"eraya/domain"
)

type Service interface {
	CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error)
	GetProducts(ctx context.Context, page, limit int64, search string, categoryIDs []int, sortBy string, minPrice, maxPrice float64, adminMode bool) ([]*domain.Product, int64, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	GetProductByID(ctx context.Context, id int64) (*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) ([]string, error)
	DeleteProduct(ctx context.Context, id int64) error
	BulkDeleteProducts(ctx context.Context, ids []int64) error
	DecrementStock(ctx context.Context, id int64, quantity int) error
	IncrementStock(ctx context.Context, id int64, quantity int) error

	// Categories
	CreateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error)
	ListCategories(ctx context.Context) ([]*domain.Category, error)
	UpdateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error)
	DeleteCategory(ctx context.Context, id int) error
	BulkDeleteCategories(ctx context.Context, ids []int) error
}

type ProductRepo interface {
	Create(ctx context.Context, product *domain.Product) (*domain.Product, error)
	List(ctx context.Context, page, limit int64, search string, categoryIDs []int, sortBy string, minPrice, maxPrice float64, adminMode bool) ([]*domain.Product, error)
	Count(ctx context.Context, search string, categoryIDs []int, minPrice, maxPrice float64, adminMode bool) (int64, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Product, error)
	FindByID(ctx context.Context, id int64) (*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) ([]string, error)
	Delete(ctx context.Context, id int64) ([]string, error)
	BulkDeleteProducts(ctx context.Context, ids []int64) ([]string, error)
	DecrementStock(ctx context.Context, id int64, quantity int) error
	IncrementStock(ctx context.Context, id int64, quantity int) error

	// Categories
	CreateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error)
	ListCategories(ctx context.Context) ([]*domain.Category, error)
	UpdateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error)
	DeleteCategory(ctx context.Context, id int) (string, error)
	BulkDeleteCategories(ctx context.Context, ids []int) ([]string, error)
}

type ProductCache interface {
	GetLatestProducts(ctx context.Context) ([]*domain.Product, error)
	SetLatestProducts(ctx context.Context, products []*domain.Product) error
}
