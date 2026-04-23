package repo

import (
	"context"
	"encoding/json"
	"eraya/domain"
	"eraya/product"
	"time"

	"github.com/redis/go-redis/v9"
)

type productCache struct {
	rdb *redis.Client
}

func NewProductCache(rdb *redis.Client) product.ProductCache {
	return &productCache{rdb: rdb}
}

const latestProductsKey = "latest_products"

func (c *productCache) GetLatestProducts(ctx context.Context) ([]*domain.Product, error) {
	val, err := c.rdb.Get(ctx, latestProductsKey).Result()
	if err != nil {
		return nil, err
	}

	var products []*domain.Product
	err = json.Unmarshal([]byte(val), &products)
	return products, err
}

func (c *productCache) SetLatestProducts(ctx context.Context, products []*domain.Product) error {
	data, err := json.Marshal(products)
	if err != nil {
		return err
	}

	// Cache for 10 minutes
	return c.rdb.Set(ctx, latestProductsKey, data, 10*time.Minute).Err()
}
