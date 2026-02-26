package repository

import (
	"Go-Microservice-Template/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// UserCache provides a caching layer for user data.
type UserCache interface {
	Get(ctx context.Context, id uuid.UUID) (*model.User, error)
	Set(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type redisUserCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisClient creates a Redis client with connection verification.
func NewRedisClient(ctx context.Context, url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	opts.PoolSize = 10
	opts.MinIdleConns = 3
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

// NewUserCache creates a new Redis-backed cache for users.
func NewUserCache(client *redis.Client, ttl time.Duration) UserCache {
	return &redisUserCache{client: client, ttl: ttl}
}

func (c *redisUserCache) key(id uuid.UUID) string {
	return fmt.Sprintf("user:%s", id.String())
}

func (c *redisUserCache) Get(ctx context.Context, id uuid.UUID) (*model.User, error) {
	if c.client == nil {
		return nil, fmt.Errorf("cache not available")
	}

	data, err := c.client.Get(ctx, c.key(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss, not an error
		}
		return nil, fmt.Errorf("cache get: %w", err)
	}

	var user model.User
	if err := json.Unmarshal(data, &user); err != nil {
		// Corrupted cache entry — delete it
		log.Warn().Err(err).Str("key", c.key(id)).Msg("corrupted cache entry, deleting")
		_ = c.Delete(ctx, id)
		return nil, nil
	}

	return &user, nil
}

func (c *redisUserCache) Set(ctx context.Context, user *model.User) error {
	if c.client == nil {
		return nil
	}

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}

	if err := c.client.Set(ctx, c.key(user.ID), data, c.ttl).Err(); err != nil {
		// Cache write failure is non-fatal — log and continue
		log.Warn().Err(err).Str("key", c.key(user.ID)).Msg("failed to write cache")
		return nil
	}

	return nil
}

func (c *redisUserCache) Delete(ctx context.Context, id uuid.UUID) error {
	if c.client == nil {
		return nil
	}

	if err := c.client.Del(ctx, c.key(id)).Err(); err != nil {
		log.Warn().Err(err).Str("key", c.key(id)).Msg("failed to delete cache")
	}

	return nil
}
