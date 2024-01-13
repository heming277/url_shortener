// storage/redis.go
package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient represents a Redis client.
type RedisClient struct {
	Client *redis.Client
}

var ErrURLNotFound = errors.New("URL not found")

// NewRedisClient creates a new Redis client.
func NewRedisClient() *RedisClient {
	// Retrieve Redis connection info from environment variables
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := os.Getenv("REDIS_DB")

	// Parse Redis DB index
	dbIndex, err := strconv.Atoi(redisDB)
	if err != nil {
		dbIndex = 0
	}

	// Initialize the Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       dbIndex,
	})

	return &RedisClient{Client: client}
}

// GetShortCodeByURL retrieves the short URL code from Redis using the original URL as the key.
func (r *RedisClient) GetShortCodeByURL(originalURL string) (string, error) {
	ctx := context.Background()

	//get through reverse mapping
	result, err := r.Client.Get(ctx, "reverse:"+originalURL).Result()
	if err == redis.Nil {
		return "", ErrURLNotFound
	} else if err != nil {
		return "", fmt.Errorf("error retrieving short code by URL: %v", err)
	}

	return result, nil
}

// StoreURLMapping stores the short URL code and the original URL in Redis.
func (r *RedisClient) StoreURLMapping(shortURLCode, originalURL string, expiration time.Duration) error {
	ctx := context.Background()

	// Use the short URL code as the key and the original URL as the value.
	err := r.Client.Set(ctx, shortURLCode, originalURL, expiration).Err()
	if err != nil {
		return fmt.Errorf("error storing URL mapping: %v", err)
	}

	// Use the original URL as the key and the short URL code as the value.---reverse map
	err = r.Client.Set(ctx, "reverse:"+originalURL, shortURLCode, expiration).Err()
	if err != nil {
		return fmt.Errorf("error storing reverse URL mapping: %v", err)
	}

	return nil
}

// RetrieveOriginalURL retrieves the original URL from Redis using the short URL code as the key.
func (r *RedisClient) RetrieveOriginalURL(shortURLCode string) (string, error) {
	ctx := context.Background()

	// Retrieve the original URL from Redis using the short URL code as the key.
	result, err := r.Client.Get(ctx, shortURLCode).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("short URL code not found")
	} else if err != nil {
		return "", fmt.Errorf("error retrieving original URL: %v", err)
	}

	return result, nil
}

func (r *RedisClient) IncrementVisitCount(shortURLCode string) error {
	ctx := context.Background()
	// Increment the visit count using a separate key with a prefix like "visits:"
	_, err := r.Client.Incr(ctx, "visits:"+shortURLCode).Result()
	if err != nil {
		return fmt.Errorf("error incrementing visit count: %v", err)
	}
	return nil
}

func (r *RedisClient) GetVisitCount(shortURLCode string) (int, error) {
	ctx := context.Background()
	// Retrieve the visit count using the key with a prefix like "visits:"
	result, err := r.Client.Get(ctx, "visits:"+shortURLCode).Result()
	if err == redis.Nil {
		// If the key does not exist, the visit count is 0
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("error retrieving visit count: %v", err)
	}
	// Convert the result to an integer
	count, err := strconv.Atoi(result)
	if err != nil {
		return 0, fmt.Errorf("error converting visit count to integer: %v", err)
	}
	return count, nil
}
