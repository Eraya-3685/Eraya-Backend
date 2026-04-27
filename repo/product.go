package repo

import (
	"context"
	"eraya/domain"
	"eraya/product"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, sales_count, average_rating, total_reviews
	`
	err = tx.QueryRowContext(ctx, query, p.Name, p.Description, p.BasePrice, p.DiscountPrice, p.DiscountPercentage, p.StockCount, p.Slug, p.IsActive).
		Scan(&p.ID, &p.CreatedAt, &p.SalesCount, &p.AverageRating, &p.TotalReviews)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Insert categories
	for _, catID := range p.CategoryIDs {
		_, err = tx.ExecContext(ctx, `INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2)`, p.ID, catID)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to link category: %w", err)
		}
	}

	// Insert images
	for i := range p.Images {
		p.Images[i].ProductID = p.ID
		_, err = tx.ExecContext(ctx, `INSERT INTO product_images (product_id, image_url, is_primary) VALUES ($1, $2, $3)`, p.Images[i].ProductID, p.Images[i].ImageURL, p.Images[i].IsPrimary)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	return p, tx.Commit()
}

func (r *productRepo) List(ctx context.Context, page, limit int64, search string, categoryIDs []int, sortBy string, minPrice, maxPrice float64) ([]*domain.Product, error) {
	var args []any
	nextParam := 1

	baseQuery := `SELECT DISTINCT p.* FROM products p`
	if len(categoryIDs) > 0 {
		baseQuery += ` JOIN product_categories pc ON p.id = pc.product_id`
	}
	baseQuery += ` WHERE p.is_active = true`

	if len(categoryIDs) > 0 {
		baseQuery += fmt.Sprintf(` AND pc.category_id = ANY($%d)`, nextParam)
		args = append(args, pq.Array(categoryIDs))
		nextParam++
	}
	if minPrice > 0 {
		baseQuery += fmt.Sprintf(` AND p.base_price >= $%d`, nextParam)
		args = append(args, minPrice)
		nextParam++
	}
	if maxPrice > 0 {
		baseQuery += fmt.Sprintf(` AND p.base_price <= $%d`, nextParam)
		args = append(args, maxPrice)
		nextParam++
	}
	if search != "" {
		baseQuery += fmt.Sprintf(` AND (p.name ILIKE $%d OR p.description ILIKE $%d)`, nextParam, nextParam+1)
		args = append(args, "%"+search+"%", "%"+search+"%")
		nextParam += 2
	}

	orderBy := "p.created_at DESC"
	switch sortBy {
	case "price_low":
		orderBy = "p.base_price ASC"
	case "price_high":
		orderBy = "p.base_price DESC"
	case "popular":
		orderBy = "p.average_rating DESC"
	case "newest":
		orderBy = "p.created_at DESC"
	}

	offset := (page - 1) * limit
	baseQuery += fmt.Sprintf(` ORDER BY %s LIMIT $%d OFFSET $%d`, orderBy, nextParam, nextParam+1)
	args = append(args, limit, offset)

	var products []*domain.Product
	err := r.db.SelectContext(ctx, &products, baseQuery, args...)
	if err != nil {
		return nil, err
	}

	// Fetch images and categories in parallel using errgroup
	g, gCtx := errgroup.WithContext(ctx)
	for i := range products {
		i := i // Capture for closure
		g.Go(func() error {
			// Fetch Images
			var images []domain.ProductImage
			if err := r.db.SelectContext(gCtx, &images, "SELECT * FROM product_images WHERE product_id = $1", products[i].ID); err == nil {
				products[i].Images = images
			}

			// Fetch Categories
			var categories []domain.Category
			query := `
				SELECT c.id, c.name, COALESCE(c.image_url, '') as image_url 
				FROM categories c
				JOIN product_categories pc ON c.id = pc.category_id
				WHERE pc.product_id = $1
			`
			if err := r.db.SelectContext(gCtx, &categories, query, products[i].ID); err == nil {
				products[i].Categories = categories
				// For frontend backward compatibility, set CategoryIDs array
				for _, c := range categories {
					products[i].CategoryIDs = append(products[i].CategoryIDs, c.ID)
				}
			}
			return nil
		})
	}

	_ = g.Wait() // Wait for all fetches to finish
	return products, nil
}

func (r *productRepo) BulkDeleteProducts(ctx context.Context, ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// 1. Fetch all image URLs for these products
	query, args, err := sqlx.In("SELECT image_url FROM product_images WHERE product_id IN (?)", ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var images []string
	err = r.db.SelectContext(ctx, &images, query, args...)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 2. Delete images and links
	delImgQuery, delImgArgs, _ := sqlx.In("DELETE FROM product_images WHERE product_id IN (?)", ids)
	_, _ = tx.ExecContext(ctx, r.db.Rebind(delImgQuery), delImgArgs...)

	delCatQuery, delCatArgs, _ := sqlx.In("DELETE FROM product_categories WHERE product_id IN (?)", ids)
	_, _ = tx.ExecContext(ctx, r.db.Rebind(delCatQuery), delCatArgs...)

	// 3. Delete products
	delProdQuery, delProdArgs, _ := sqlx.In("DELETE FROM products WHERE id IN (?)", ids)
	_, err = tx.ExecContext(ctx, r.db.Rebind(delProdQuery), delProdArgs...)
	if err != nil {
		return nil, err
	}

	return images, tx.Commit()
}

func (r *productRepo) DecrementStock(ctx context.Context, id int64, quantity int) error {
	query := `
		UPDATE products 
		SET stock_count = stock_count - $1, 
		    sales_count = sales_count + $1 
		WHERE id = $2 AND stock_count >= $1
	`
	res, err := r.db.ExecContext(ctx, query, quantity, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient stock for product %d", id)
	}
	return nil
}

func (r *productRepo) IncrementStock(ctx context.Context, id int64, quantity int) error {
	query := `
		UPDATE products 
		SET stock_count = stock_count + $1, 
		    sales_count = sales_count - $1 
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, quantity, id)
	return err
}

func (r *productRepo) Count(ctx context.Context, search string, categoryIDs []int, minPrice, maxPrice float64) (int64, error) {
	var args []interface{}
	nextParam := 1

	baseQuery := `SELECT COUNT(DISTINCT p.id) FROM products p`
	if len(categoryIDs) > 0 {
		baseQuery += ` JOIN product_categories pc ON p.id = pc.product_id`
	}
	baseQuery += ` WHERE p.is_active = true`

	if len(categoryIDs) > 0 {
		baseQuery += fmt.Sprintf(` AND pc.category_id = ANY($%d)`, nextParam)
		args = append(args, pq.Array(categoryIDs))
		nextParam++
	}
	if minPrice > 0 {
		baseQuery += fmt.Sprintf(` AND p.base_price >= $%d`, nextParam)
		args = append(args, minPrice)
		nextParam++
	}
	if maxPrice > 0 {
		baseQuery += fmt.Sprintf(` AND p.base_price <= $%d`, nextParam)
		args = append(args, maxPrice)
		nextParam++
	}
	if search != "" {
		baseQuery += fmt.Sprintf(` AND (p.name ILIKE $%d OR p.description ILIKE $%d)`, nextParam, nextParam+1)
		args = append(args, "%"+search+"%", "%"+search+"%")
		nextParam += 2
	}

	var count int64
	err := r.db.GetContext(ctx, &count, baseQuery, args...)
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
		SET name = $1, description = $2, base_price = $3, 
		    discount_price = $4, discount_percentage = $5, 
		    stock_count = $6, slug = $7, is_active = $8
		WHERE id = $9
	`
	_, err = tx.ExecContext(ctx, query, p.Name, p.Description, p.BasePrice, p.DiscountPrice, p.DiscountPercentage, p.StockCount, p.Slug, p.IsActive, p.ID)
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
				_, err = tx.ExecContext(ctx, `INSERT INTO product_images (product_id, image_url, is_primary) VALUES ($1, $2, $3)`, p.ID, img.ImageURL, img.IsPrimary)
				if err != nil {
					return nil, err
				}
			} else {
				_, err = tx.ExecContext(ctx, `UPDATE product_images SET is_primary = $1 WHERE image_url = $2 AND product_id = $3`, img.IsPrimary, img.ImageURL, p.ID)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// 3. Sync categories
	if p.CategoryIDs != nil {
		_, err = tx.ExecContext(ctx, "DELETE FROM product_categories WHERE product_id = $1", p.ID)
		if err != nil {
			return nil, err
		}
		for _, catID := range p.CategoryIDs {
			_, err = tx.ExecContext(ctx, "INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2)", p.ID, catID)
			if err != nil {
				return nil, err
			}
		}
	}

	return orphanedURLs, tx.Commit()
}

func (r *productRepo) Delete(ctx context.Context, id int64) ([]string, error) {
	// 1. Fetch all images for this product before deleting
	var images []string
	err := r.db.SelectContext(ctx, &images, "SELECT image_url FROM product_images WHERE product_id = $1", id)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 2. Delete links first
	_, err = tx.ExecContext(ctx, "DELETE FROM product_images WHERE product_id = $1", id)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM product_categories WHERE product_id = $1", id)
	if err != nil {
		return nil, err
	}

	// 3. Delete product
	_, err = tx.ExecContext(ctx, "DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return nil, err
	}

	return images, tx.Commit()
}

// Categories
func (r *productRepo) CreateCategory(ctx context.Context, c *domain.Category) (*domain.Category, error) {
	// Check for duplicate name (case-insensitive)
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE LOWER(name) = LOWER($1))`, c.Name).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("category '%s' already exists", c.Name)
	}

	query := `INSERT INTO categories (name, image_url) VALUES ($1, $2) RETURNING id, name, image_url, 0 as product_count`
	var created domain.Category
	err = r.db.GetContext(ctx, &created, query, c.Name, c.ImageURL)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *productRepo) ListCategories(ctx context.Context) ([]*domain.Category, error) {
	var categories []*domain.Category
	query := `
		SELECT c.id, c.name, COALESCE(c.image_url, '') as image_url, COUNT(pc.product_id) as product_count 
		FROM categories c 
		LEFT JOIN product_categories pc ON c.id = pc.category_id 
		GROUP BY c.id, c.name, c.image_url 
		ORDER BY c.name ASC`
	err := r.db.SelectContext(ctx, &categories, query)
	return categories, err
}

func (r *productRepo) UpdateCategory(ctx context.Context, c *domain.Category) (*domain.Category, error) {
	// Check for duplicate name (case-insensitive), excluding current category
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE LOWER(name) = LOWER($1) AND id != $2)`, c.Name, c.ID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("category '%s' already exists", c.Name)
	}

	query := `UPDATE categories SET name = $1, image_url = $2 WHERE id = $3 RETURNING id, name, image_url, 0 as product_count`
	var updated domain.Category
	err = r.db.GetContext(ctx, &updated, query, c.Name, c.ImageURL, c.ID)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (r *productRepo) DeleteCategory(ctx context.Context, id int) (string, error) {
	// 1. Fetch image URL before deleting
	var imageURL string
	_ = r.db.GetContext(ctx, &imageURL, "SELECT COALESCE(image_url, '') FROM categories WHERE id = $1", id)

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	// First remove all product links to avoid FK constraint
	_, err = tx.ExecContext(ctx, "DELETE FROM product_categories WHERE category_id = $1", id)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	// Then delete the category
	_, err = tx.ExecContext(ctx, "DELETE FROM categories WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	return imageURL, tx.Commit()
}

func (r *productRepo) BulkDeleteCategories(ctx context.Context, ids []int) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// 1. Fetch image URLs
	query, args, err := sqlx.In("SELECT COALESCE(image_url, '') FROM categories WHERE id IN (?)", ids)
	if err != nil {
		return nil, err
	}
	var images []string
	err = r.db.SelectContext(ctx, &images, r.db.Rebind(query), args...)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Remove product_categories links first
	unlinkQuery, unlinkArgs, _ := sqlx.In("DELETE FROM product_categories WHERE category_id IN (?)", ids)
	_, _ = tx.ExecContext(ctx, r.db.Rebind(unlinkQuery), unlinkArgs...)

	// Then delete the categories
	query, args, _ = sqlx.In("DELETE FROM categories WHERE id IN (?)", ids)
	_, err = tx.ExecContext(ctx, r.db.Rebind(query), args...)
	if err != nil {
		return nil, err
	}

	return images, tx.Commit()
}
