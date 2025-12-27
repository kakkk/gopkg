package dlock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisLock struct {
	client    *redis.Client
	lockKey   string
	lockValue string // UUID 值，用于安全释放锁
	mu        sync.Mutex
	unlocked  bool // 标记是否已释放
}

func (l *redisLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.unlocked {
		return nil // 幂等性：已经释放的锁再次释放不报错
	}

	// 使用 Lua 脚本确保原子性：只有锁的值匹配时才删除
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, l.client, []string{l.lockKey}, l.lockValue).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 锁已经过期或被删除
			l.unlocked = true
			return nil
		}
		return fmt.Errorf("redis error: %w", err)
	}

	if result == 0 {
		// 锁的值不匹配或锁不存在
		return ErrLockNotHeld
	}

	l.unlocked = true
	return nil
}

func newRedisLocker(client *redis.Client) *redisLocker {
	return &redisLocker{
		client: client,
	}
}

type redisLocker struct {
	client *redis.Client
}

func (r *redisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	// 参数校验
	if err := validateKeyAndTTL(key, ttl); err != nil {
		return nil, err
	}

	value := lockValue()

	success, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}
	if !success {
		return nil, ErrLockAlreadyHeld
	}

	return &redisLock{
		client:    r.client,
		lockKey:   key,
		lockValue: value,
	}, nil
}

func (r *redisLocker) AcquireWithRetry(ctx context.Context, key string, ttl time.Duration, maxRetry int64, interval time.Duration) (Lock, error) {
	// 参数校验
	if err := validateKeyAndTTL(key, ttl); err != nil {
		return nil, err
	}
	if maxRetry < 0 {
		maxRetry = 0
	}

	for i := int64(0); i <= maxRetry; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		lock, err := r.Acquire(ctx, key, ttl)
		if err == nil {
			return lock, nil
		}

		// 等待后重试
		select {
		case <-time.After(interval):
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, ErrLockNotAcquired
}
