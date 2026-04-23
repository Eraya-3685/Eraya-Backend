package product

import (
	"eraya/domain"
)

type service struct {
	repo  ProductRepo
	cache ProductCache
}

func NewService(repo ProductRepo, cache ProductCache) Service {
	return &service{
		repo:  repo,
		cache: cache,
	}
}

func (s *service) CreateProduct(product *domain.Product) (*domain.Product, error) {
	p, err := s.repo.Create(product)
	if err == nil {
		// Invalidate cache asynchronously (first 100 products)
		go func() {
			products, _ := s.repo.List(1, 100)
			s.cache.SetLatestProducts(products)
		}()
	}
	return p, err
}

func (s *service) GetProducts(page, limit int64) ([]*domain.Product, int64, error) {
	products, err := s.repo.List(page, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count()
	if err != nil {
		return nil, 0, err
	}

	return products, count, nil
}

func (s *service) GetProductBySlug(slug string) (*domain.Product, error) {
	return s.repo.FindBySlug(slug)
}

func (s *service) GetProductByID(id int64) (*domain.Product, error) {
	return s.repo.FindByID(id)
}

func (s *service) UpdateProduct(p *domain.Product) error {
	err := s.repo.Update(p)
	if err == nil {
		go s.invalidateCache()
	}
	return err
}

func (s *service) DeleteProduct(id int64) error {
	err := s.repo.Delete(id)
	if err == nil {
		go s.invalidateCache()
	}
	return err
}

func (s *service) invalidateCache() {
	products, _ := s.repo.List(1, 100)
	s.cache.SetLatestProducts(products)
}
