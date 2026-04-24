package repo

import (
	"context"
	"eraya/domain"
	"eraya/product"
	"fmt"

	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

type productRepo struct {
	db *sqlx.DB
}

func NewProductRepo(db *sqlx.DB) product.ProductRepo {
	return &productRepo{db: db}
}

func (r *productRepo) Create(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO products (name, description, base_price, discount_price, discount_percentage, stock_count, slug, is_active)
		VALUES (:name, :description, :base_price, :discount_price, :discount_percentage, :stock_count, :slug, :is_active)
		RETURNING id, created_at, sales_count, average_rating, total_reviews
	`
	rows, err := sqlx.NamedQueryContext(ctx, tx, query, p)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create product: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&p.ID, &p.CreatedAt, &p.SalesCount, &p.AverageRating, &p.TotalReviews)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	rows.Close()

	// Insert images
	for i := range p.Images {
		p.Images[i].ProductID = p.ID
		_, err = tx.NamedExecContext(ctx, `INSERT INTO product_images (product_id, image_url, is_primary) VALUES (:product_id, :image_url, :is_primary)`, p.Images[i])
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	return p, tx.Commit()
}

func (r *productRepo) List(ctx context.Context, page, limit int64, search string) ([]*domain.Product, error) {
	offset := (page - 1) * limit
	query := `SELECT * FROM products WHERE is_active = true`
	args := []interface{}{limit, offset}

	if search != "" {
		query += ` AND (name ILIKE $3 OR description ILIKE $3)`
		args = append(args, "%"+search+"%")
	}

	query += ` ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	var products []*domain.Product
	err := r.db.SelectContext(ctx, &products, query, args...)
	if err != nil {
		return nil, err
	}

	// Fetch images in parallel using errgroup
	g, gCtx := errgroup.WithContext(ctx)
	for i := range products {
		i := i // Capture for closure
		g.Go(func() error {
			var images []domain.ProductImage
			err := r.db.SelectContext(gCtx, &images, "SELECT * FROM product_images WHERE product_id = $1", products[i].ID)
			if err == nil {
				products[i].Images = images
			}
			return nil // Don't fail the whole list if one image fetch fails
		})
	}

	_ = g.Wait() // Wait for all fetches to finish
	return products, nil
}

func (r *productRepo) Count(ctx context.Context, search string) (int64, error) {
	query := `SELECT COUNT(*) FROM products WHERE is_active = true`
	var args []interface{}
	if search != "" {
		query += ` AND (name ILIKE $1 OR description ILIKE $1)`
		args = append(args, "%"+search+"%")
	}

	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	return count, err
}

func (r *productRepo) FindBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	query := `SELECT * FROM products WHERE slug = $1 AND is_active = true LIMIT 1`
	var p domain.Product
	err := r.db.GetContext(ctx, &p, query, slug)
	if err != nil {
		return nil, err
	}

	// Fetch all images
	var images []domain.ProductImage
	err = r.db.SelectContext(ctx, &images, "SELECT * FROM product_images WHERE product_id = $1", p.ID)
	if err == nil {
		p.Images = images
	}

	return &p, nil
}

func (r *productRepo) FindByID(ctx context.Context, id int64) (*domain.Product, error) {
	query := `SELECT * FROM products WHERE id = $1 AND is_active = true LIMIT 1`
	var p domain.Product
	err := r.db.GetContext(ctx, &p, query, id)
	if err != nil {
		return nil, err
	}

	var images []domain.ProductImage
	err = r.db.SelectContext(ctx, &images, "SELECT * FROM product_images WHERE product_id = $1", p.ID)
	if err == nil {
		p.Images = images
	}

	return &p, nil
}

func (r *productRepo) Update(ctx context.Context, p *domain.Product) ([]string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Update main product info
	query := `
		UPDATE products 
		SET name = :name, description = :description, base_price = :base_price, 
		    discount_price = :discount_price, discount_percentage = :discount_percentage, 
		    stock_count = :stock_count, slug = :slug, is_active = :is_active
		WHERE id = :id
	`
	_, err = tx.NamedExecContext(ctx, query, p)
	if err != nil {
		return nil, err
	}

	var orphanedURLs []string

	// 2. Sync images if provided
	if p.Images != nil {
		// Get existing images from DB
		var currentImages []domain.ProductImage
		err = tx.SelectContext(ctx, &currentImages, "SELECT * FROM product_images WHERE product_id = $1", p.ID)
		if err != nil {
			return nil, err
		}

		// Identify images to delete
		newImageURLs := make(map[string]bool)
		for _, img := range p.Images {
			newImageURLs[img.ImageURL] = true
		}

		for _, oldImg := range currentImages {
			if !newImageURLs[oldImg.ImageURL] {
				orphanedURLs = append(orphanedURLs, oldImg.ImageURL)
				_, err = tx.ExecContext(ctx, "DELETE FROM product_images WHERE id = $1", oldImg.ID)
				if err != nil {
					return nil, err
				}
			}
		}

		// Add new images (only those not already in DB)
		existingURLs := make(map[string]bool)
		for _, oldImg := range currentImages {
			existingURLs[oldImg.ImageURL] = true
		}

		for _, img := range p.Images {
			if !existingURLs[img.ImageURL] {
				img.ProductID = p.ID
				_, err = tx.NamedExecContext(ctx, `INSERT INTO product_images (product_id, image_url, is_primary) VALUES (:product_id, :image_url, :is_primary)`, img)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return orphanedURLs, tx.Commit()
}

func (r *productRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM products WHERE id = $1", id)
	return err
}

// Categories
func (r *productRepo) CreateCategory(ctx context.Context, c *domain.Category) (*domain.Category, error) {
	query := `INSERT INTO categories (name) VALUES ($1) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, c.Name).Scan(&c.ID)
	return c, err
}

func (r *productRepo) ListCategories(ctx context.Context) ([]*domain.Category, error) {
	var categories []*domain.Category
	err := r.db.SelectContext(ctx, &categories, "SELECT * FROM categories ORDER BY name ASC")
	return categories, err
}
