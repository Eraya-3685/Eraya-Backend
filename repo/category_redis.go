package repo

import (
	"context"
	"encoding/json"
	"eraya/domain"
	"eraya/product"
	"time"

	"github.com/redis/go-redis/v9"
)

const categoriesCacheKey = "categories:all"
const categoriesTTL = 30 * time.Minute

type categoryCache struct {
	rdb *redis.Client
}

func NewCategoryCache(rdb *redis.Client) product.CategoryCache {
	return &categoryCache{rdb: rdb}
}

func (c *categoryCache) GetCategories(ctx context.Context) ([]*domain.Category, error) {
	val, err := c.rdb.Get(ctx, categoriesCacheKey).Result()
	if err != nil {
		return nil, err
	}
	var cats []*domain.Category
	err = json.Unmarshal([]byte(val), &cats)
	return cats, err
}

func (c *categoryCache) SetCategories(ctx context.Context, cats []*domain.Category) error {
	data, err := json.Marshal(cats)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, categoriesCacheKey, data, categoriesTTL).Err()
}

func (c *categoryCache) InvalidateCategories(ctx context.Context) {
	c.rdb.Del(ctx, categoriesCacheKey)
}
