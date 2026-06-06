package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"
)

type Category struct {
	ID           int    `json:"id" db:"id"`
	Name         string  `json:"name" db:"name"`
	ImageURL     *string `json:"image_url" db:"image_url"`
	ProductCount int     `json:"product_count" db:"product_count"`
}

type Product struct {
	ID                 int64          `json:"id" db:"id"`
	Name               string         `json:"name" db:"name"`
	Description        *string        `json:"description" db:"description"`
	BasePrice          float64        `json:"base_price" db:"base_price"`
	DiscountPrice      *float64       `json:"discount_price" db:"discount_price"`
	DiscountPercentage *int           `json:"discount_percentage" db:"discount_percentage"`
	StockCount         int            `json:"stock_count" db:"stock_count"`
	CategoryID         *int           `json:"category_id,omitempty" db:"category_id"` 
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	Slug               string         `json:"slug" db:"slug"`
	IsActive           bool           `json:"is_active" db:"is_active"`
	SalesCount         int            `json:"sales_count" db:"sales_count"`
	AverageRating      float64        `json:"average_rating" db:"average_rating"`
	TotalReviews       int            `json:"total_reviews" db:"total_reviews"`
	ImageUrl           string         `json:"image_url,omitempty" db:"image_url"`
	Colors             pq.StringArray     `json:"colors" db:"colors" swaggertype:"array,string"`
	Sizes              pq.StringArray     `json:"sizes" db:"sizes" swaggertype:"array,string"`
	VariationStock     VariationStockList `json:"variation_stock" db:"variation_stock" swaggertype:"string"`

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

type VariationStock struct {
	Color string `json:"color"`
	Size  string `json:"size"`
	Stock int    `json:"stock"`
}

type VariationStockList []VariationStock

func (vl *VariationStockList) Scan(value interface{}) error {
	if value == nil {
		*vl = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, vl)
}

func (vl VariationStockList) Value() (driver.Value, error) {
	if vl == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(vl)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}
