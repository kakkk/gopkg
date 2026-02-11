package cachex

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	cli *redis.Client
}

func NewRedisCacher(cli *redis.Client) Cacher {
	return &redisCache{
		cli: cli,
	}
}

func (r *redisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.cli.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("redis error: %w", err)
	}
	return val, nil
}

func (r *redisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	values, err := r.cli.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}
	result := make(map[string][]byte, len(keys))
	for i, key := range keys {
		val, ok := values[i].(string)
		if !ok || val == "" {
			result[key] = nil
			continue
		}
		result[key] = stringToBytes(val)
	}
	return result, nil
}

func (r *redisCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	err := r.cli.Set(ctx, key, val, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

func (r *redisCache) MSet(ctx context.Context, kvs map[string][]byte, ttl time.Duration) error {
	pipe := r.cli.Pipeline()
	for k, v := range kvs {
		pipe.Set(ctx, k, v, ttl)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

func (r *redisCache) Delete(ctx context.Context, key string) error {
	err := r.cli.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

func (r *redisCache) MDelete(ctx context.Context, keys []string) error {
	err := r.cli.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}
