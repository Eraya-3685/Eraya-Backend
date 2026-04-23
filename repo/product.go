package repo

import (
	"eraya/domain"
	"eraya/product"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type productRepo struct {
	db *sqlx.DB
}

func NewProductRepo(db *sqlx.DB) product.ProductRepo {
	return &productRepo{db: db}
}

func (r *productRepo) Create(p *domain.Product) (*domain.Product, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO products (name, description, base_price, discount_price, discount_percentage, stock_count, slug, is_active)
		VALUES (:name, :description, :base_price, :discount_price, :discount_percentage, :stock_count, :slug, :is_active)
		RETURNING id, created_at, sales_count, average_rating, total_reviews
	`
	rows, err := tx.NamedQuery(query, p)
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
		_, err = tx.NamedExec(`INSERT INTO product_images (product_id, image_url, is_primary) VALUES (:product_id, :image_url, :is_primary)`, p.Images[i])
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	return p, tx.Commit()
}

func (r *productRepo) List(page, limit int64) ([]*domain.Product, error) {
	offset := (page - 1) * limit
	query := `SELECT * FROM products WHERE is_active = true ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	var products []*domain.Product
	err := r.db.Select(&products, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Fetch primary images for list
	for i, p := range products {
		var images []domain.ProductImage
		r.db.Select(&images, "SELECT * FROM product_images WHERE product_id = $1", p.ID)
		products[i].Images = images
	}

	return products, nil
}

func (r *productRepo) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM products WHERE is_active = true`
	var count int64
	err := r.db.Get(&count, query)
	return count, err
}

func (r *productRepo) FindBySlug(slug string) (*domain.Product, error) {
	query := `SELECT * FROM products WHERE slug = $1 AND is_active = true LIMIT 1`
	var p domain.Product
	err := r.db.Get(&p, query, slug)
	if err != nil {
		return nil, err
	}

	// Fetch all images
	var images []domain.ProductImage
	r.db.Select(&images, "SELECT * FROM product_images WHERE product_id = $1", p.ID)
	p.Images = images

	return &p, nil
}

func (r *productRepo) FindByID(id int64) (*domain.Product, error) {
	query := `SELECT * FROM products WHERE id = $1 AND is_active = true LIMIT 1`
	var p domain.Product
	err := r.db.Get(&p, query, id)
	if err != nil {
		return nil, err
	}

	var images []domain.ProductImage
	r.db.Select(&images, "SELECT * FROM product_images WHERE product_id = $1", p.ID)
	p.Images = images

	return &p, nil
}

func (r *productRepo) Update(p *domain.Product) error {
	query := `
		UPDATE products 
		SET name = :name, description = :description, base_price = :base_price, 
		    discount_price = :discount_price, discount_percentage = :discount_percentage, 
		    stock_count = :stock_count, slug = :slug, is_active = :is_active
		WHERE id = :id
	`
	_, err := r.db.NamedExec(query, p)
	return err
}

func (r *productRepo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM products WHERE id = $1", id)
	return err
}
