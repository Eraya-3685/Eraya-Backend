package redis

import (
	"context"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(redisURL string) (*redis.Client, error) {
	var opts *redis.Options
	var err error

	if strings.HasPrefix(redisURL, "redis://") || strings.HasPrefix(redisURL, "rediss://") {
		opts, err = redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("Failed to parse Redis URL: %v", err)
			return nil, err
		}
	} else {
		opts = &redis.Options{
			Addr: redisURL,
			DB:   0,
		}
	}

	redisDB := redis.NewClient(opts)

	ctx := context.Background()
	_, err = redisDB.Ping(ctx).Result()
	// Silent connection
	return redisDB, nil
}
