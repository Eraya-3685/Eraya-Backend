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
	query := `
		INSERT INTO products (name, description, base_price, discount_price, discount_percentage, stock_count, slug, is_active)
		VALUES (:name, :description, :base_price, :discount_price, :discount_percentage, :stock_count, :slug, :is_active)
		RETURNING id, created_at, sales_count, average_rating, total_reviews
	`
	rows, err := r.db.NamedQuery(query, p)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&p.ID, &p.CreatedAt, &p.SalesCount, &p.AverageRating, &p.TotalReviews)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (r *productRepo) List(page, limit int64) ([]*domain.Product, error) {
	offset := (page - 1) * limit
	query := `SELECT * FROM products WHERE is_active = true ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	var products []*domain.Product
	err := r.db.Select(&products, query, limit, offset)
	return products, err
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
	return &p, err
}

func (r *productRepo) FindByID(id int64) (*domain.Product, error) {
	query := `SELECT * FROM products WHERE id = $1 AND is_active = true LIMIT 1`
	var p domain.Product
	err := r.db.Get(&p, query, id)
	return &p, err
}
