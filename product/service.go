package product

import (
	"context"
	"eraya/domain"
	"eraya/infra/storage"
	"log/slog"
	"time"
)

type service struct {
	repo          ProductRepo
	cache         ProductCache
	categoryCache CategoryCache
	storage       *storage.StorageService
}

func NewService(repo ProductRepo, cache ProductCache, catCache CategoryCache, storage *storage.StorageService) Service {
	return &service{
		repo:          repo,
		cache:         cache,
		categoryCache: catCache,
		storage:       storage,
	}
}

func (s *service) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	p, err := s.repo.Create(ctx, product)
	if err == nil {
		// Invalidate cache asynchronously with timeout
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.invalidateCache(ctx)
		}()
	}
	return p, err
}

func (s *service) GetProducts(ctx context.Context, page, limit int64, search string, categoryIDs []int, sortBy string, minPrice, maxPrice float64, adminMode bool) ([]*domain.Product, int64, error) {
	products, err := s.repo.List(ctx, page, limit, search, categoryIDs, sortBy, minPrice, maxPrice, adminMode)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(ctx, search, categoryIDs, minPrice, maxPrice, adminMode)
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

func (s *service) UpdateProduct(ctx context.Context, p *domain.Product) ([]string, error) {
	orphanedURLs, err := s.repo.Update(ctx, p)
	if err == nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			s.invalidateCache(ctx)
			for _, url := range orphanedURLs {
				if url != "" {
					s.storage.DeleteFile(url)
				}
			}
		}()
	}
	return orphanedURLs, err
}

func (s *service) DeleteProduct(ctx context.Context, id int64) error {
	imageURLs, err := s.repo.Delete(ctx, id)
	if err == nil {
		go s.invalidateCache(context.Background())
		// Cleanup images from storage
		for _, url := range imageURLs {
			if url != "" {
				go s.storage.DeleteFile(url)
			}
		}
	}
	return err
}

func (s *service) BulkDeleteProducts(ctx context.Context, ids []int64) error {
	imageURLs, err := s.repo.BulkDeleteProducts(ctx, ids)
	if err == nil {
		go s.invalidateCache(context.Background())
		// Cleanup images from storage
		for _, url := range imageURLs {
			if url != "" {
				go s.storage.DeleteFile(url)
			}
		}
	}
	return err
}

func (s *service) DecrementStock(ctx context.Context, id int64, quantity int) error {
	err := s.repo.DecrementStock(ctx, id, quantity)
	if err == nil {
		// Stock change should also invalidate cache if we show stock count in lists
		go s.invalidateCache(context.Background())
	}
	return err
}

func (s *service) IncrementStock(ctx context.Context, id int64, quantity int) error {
	err := s.repo.IncrementStock(ctx, id, quantity)
	if err == nil {
		go s.invalidateCache(context.Background())
	}
	return err
}

func (s *service) CreateCategory(ctx context.Context, c *domain.Category) (*domain.Category, error) {
	result, err := s.repo.CreateCategory(ctx, c)
	if err == nil {
		go s.categoryCache.InvalidateCategories(context.Background())
	}
	return result, err
}

func (s *service) ListCategories(ctx context.Context) ([]*domain.Category, error) {
	// Try cache first
	if cats, err := s.categoryCache.GetCategories(ctx); err == nil {
		return cats, nil
	}
	// Fallback to DB
	cats, err := s.repo.ListCategories(ctx)
	if err == nil {
		go s.categoryCache.SetCategories(context.Background(), cats)
	}
	return cats, err
}

func (s *service) UpdateCategory(ctx context.Context, category *domain.Category) (*domain.Category, error) {
	oldCatList, _ := s.repo.ListCategories(ctx)
	var oldImg string
	for _, c := range oldCatList {
		if c.ID == category.ID {
			if c.ImageURL != nil {
				oldImg = *c.ImageURL
			}
			break
		}
	}

	updated, err := s.repo.UpdateCategory(ctx, category)
	if err == nil {
		go s.categoryCache.InvalidateCategories(context.Background())
		if oldImg != "" {
			newImg := ""
			if category.ImageURL != nil {
				newImg = *category.ImageURL
			}
			if oldImg != newImg {
				go s.storage.DeleteFile(oldImg)
			}
		}
	}
	return updated, err
}

func (s *service) DeleteCategory(ctx context.Context, id int) error {
	imageURL, err := s.repo.DeleteCategory(ctx, id)
	if err == nil {
		go s.categoryCache.InvalidateCategories(context.Background())
		if imageURL != "" {
			go s.storage.DeleteFile(imageURL)
		}
	}
	return err
}

func (s *service) BulkDeleteCategories(ctx context.Context, ids []int) error {
	imageURLs, err := s.repo.BulkDeleteCategories(ctx, ids)
	if err == nil {
		go s.categoryCache.InvalidateCategories(context.Background())
		for _, url := range imageURLs {
			if url != "" {
				go s.storage.DeleteFile(url)
			}
		}
	}
	return err
}

func (s *service) invalidateCache(ctx context.Context) {
	products, err := s.repo.List(ctx, 1, 100, "", []int{}, "newest", 0, 0, false)
	if err != nil {
		slog.Error("Failed to fetch products for cache invalidation", "error", err)
		return
	}
	s.cache.SetLatestProducts(ctx, products)
}
