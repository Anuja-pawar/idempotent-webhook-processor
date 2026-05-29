package repository

import (
	"context"
	"time"

	"idempotent-webhook-processor/internal/domain"

	"github.com/redis/go-redis/v9"
)

type redisRepository struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisRepository instantiates a new Redis storage layer for idempotency tracking.
func NewRedisRepository(client *redis.Client, lockTTL time.Duration) domain.IdempotencyRepository {
	return &redisRepository{
		client: client,
		ttl:    lockTTL,
	}
}

func (r *redisRepository) SetLock(ctx context.Context, key string) (bool, error) {
	// NX: Set if Not Exists (makes the lock acquisition completely atomic)
	success, err := r.client.SetNX(ctx, "lock:"+key, "processing", r.ttl).Result()
	if err != nil {
		return false, err
	}
	return success, nil
}

func (r *redisRepository) ReleaseLock(ctx context.Context, key string) error {
	return r.client.Del(ctx, "lock:"+key).Err()
}
