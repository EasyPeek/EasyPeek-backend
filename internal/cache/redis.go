package cache

import (
	"context"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(cfg config.RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.Database,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
	}, nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}
