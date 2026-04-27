package domain

import "time"

type Category struct {
	ID           int    `json:"id" db:"id"`
	Name         string  `json:"name" db:"name"`
	ImageURL     *string `json:"image_url" db:"image_url"`
	ProductCount int     `json:"product_count" db:"product_count"`
}

type Product struct {
	ID                 int64     `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	Description        *string   `json:"description" db:"description"`
	BasePrice          float64   `json:"base_price" db:"base_price"`
	DiscountPrice      *float64  `json:"discount_price" db:"discount_price"`
	DiscountPercentage *int      `json:"discount_percentage" db:"discount_percentage"`
	StockCount         int       `json:"stock_count" db:"stock_count"`
	CategoryID         *int      `json:"category_id,omitempty" db:"category_id"` 
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	Slug               string    `json:"slug" db:"slug"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	SalesCount         int       `json:"sales_count" db:"sales_count"`
	AverageRating      float64   `json:"average_rating" db:"average_rating"`
	TotalReviews       int       `json:"total_reviews" db:"total_reviews"`
	ImageUrl           string    `json:"image_url,omitempty" db:"image_url"`

	CategoryIDs        []int          `json:"category_ids,omitempty" db:"-"`
	Categories         []Category      `json:"categories,omitempty" db:"-"`
	Images             []ProductImage `json:"images,omitempty" db:"-"`
}

type ProductImage struct {
	ID        int64  `json:"id" db:"id"`
	ProductID int64  `json:"product_id" db:"product_id"`
	ImageURL  string `json:"image_url" db:"image_url"`
	IsPrimary bool   `json:"is_primary" db:"is_primary"`
}
