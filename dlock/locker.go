package dlock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Lock interface {
	Unlock(ctx context.Context) error
}

type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error)
	AcquireWithRetry(ctx context.Context, key string, ttl time.Duration, maxRetry int64, interval time.Duration) (Lock, error)
}

// NewDatabaseLocker 基于Redis的分布式锁，SetNX加锁，LUA脚本释放
func NewRedisLocker(cli *redis.Client) Locker {
	return newRedisLocker(cli)
}

// NewDatabaseLocker 基于数据库的分布式锁，唯一索引+Insert实现，兼容MySQL、PostgreSQL、SQLite
func NewDatabaseLocker(db *gorm.DB, table string) Locker {
	return newDatabaseLocker(db, table)
}
