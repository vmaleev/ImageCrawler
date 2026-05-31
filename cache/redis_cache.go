package cache

import (
	"ImageCrawler/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
}

var ctx = context.Background()

func NewRedisCache() (*RedisCache, error) {
	redisAddr := strings.TrimSpace(os.Getenv("REDIS_ADDRESS"))
	if redisAddr == "" {
		return nil, errors.New("REDIS_ADDRESS environment variable is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	pingCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to Redis at %q: %w", redisAddr, err)
	}

	return &RedisCache{client: client}, nil
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
