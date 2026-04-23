package product

import "eraya/domain"

type Service interface {
	CreateProduct(product *domain.Product) (*domain.Product, error)
	GetProducts(page, limit int64) ([]*domain.Product, int64, error)
	GetProductBySlug(slug string) (*domain.Product, error)
	GetProductByID(id int64) (*domain.Product, error)
	UpdateProduct(product *domain.Product) error
	DeleteProduct(id int64) error
}

type ProductRepo interface {
	Create(product *domain.Product) (*domain.Product, error)
	List(page, limit int64) ([]*domain.Product, error)
	Count() (int64, error)
	FindBySlug(slug string) (*domain.Product, error)
	FindByID(id int64) (*domain.Product, error)
	Update(product *domain.Product) error
	Delete(id int64) error
}

type ProductCache interface {
	GetLatestProducts() ([]*domain.Product, error)
	SetLatestProducts(products []*domain.Product) error
}
