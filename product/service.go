package product

import (
	"context"
	"eraya/domain"
	"log/slog"
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

func (s *service) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	p, err := s.repo.Create(ctx, product)
	if err == nil {
		// Invalidate cache asynchronously
		go s.invalidateCache(context.Background())
	}
	return p, err
}

func (s *service) GetProducts(ctx context.Context, page, limit int64) ([]*domain.Product, int64, error) {
	products, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return products, count, nil
}

func (s *service) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	return s.repo.FindBySlug(ctx, slug)
}

func (s *service) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) UpdateProduct(ctx context.Context, p *domain.Product) error {
	err := s.repo.Update(ctx, p)
	if err == nil {
		go s.invalidateCache(context.Background())
	}
	return err
}

func (s *service) DeleteProduct(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if err == nil {
		go s.invalidateCache(context.Background())
	}
	return err
}

func (s *service) invalidateCache(ctx context.Context) {
	products, err := s.repo.List(ctx, 1, 100)
	if err != nil {
		slog.Error("Failed to fetch products for cache invalidation", "error", err)
		return
	}
	s.cache.SetLatestProducts(ctx, products)
}
