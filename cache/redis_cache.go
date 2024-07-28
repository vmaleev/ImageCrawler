package cache

import (
	"ImageCrawler/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
)

type RedisCache struct {
	client *redis.Client
}

var ctx = context.Background()

func NewRedisCache() *RedisCache {
	redisAddr := os.Getenv("REDIS_ADDRESS")
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	return &RedisCache{client: client}
}

func (r *RedisCache) Get(key string) (models.Metadata, bool) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return models.Metadata{}, false
	} else if err != nil {
		fmt.Printf("Failed to get key from Redis: %v\n", err)
		return models.Metadata{}, false
	}

	var metadata models.Metadata
	err = json.Unmarshal([]byte(val), &metadata)
	if err != nil {
		fmt.Printf("Failed to unmarshal metadata: %v\n", err)
		return models.Metadata{}, false
	}

	return metadata, true
}

func (r *RedisCache) Set(key string, val models.Metadata) {
	data, err := json.Marshal(val)
	if err != nil {
		fmt.Printf("Failed to marshal metadata: %v\n", err)
		return
	}

	err = r.client.Set(ctx, key, data, 0).Err()
	if err != nil {
		fmt.Printf("Failed to set key in Redis: %v\n", err)
	}
}

func (r *RedisCache) Exists(key string) bool {
	_, err := r.client.Get(ctx, key).Result()
	return err == nil
}

func (r *RedisCache) Invalidate(key string) {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		fmt.Printf("Failed to invalidate key in Redis: %v\n", err)
	}
}
