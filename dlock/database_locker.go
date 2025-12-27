package dlock

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"
)

type lockModel struct {
	ID         uint      `gorm:"column:id"`
	LockKey    string    `gorm:"column:lock_key"`
	LockValue  string    `gorm:"column:lock_value"`
	ExpireTime time.Time `gorm:"column:expire_time"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

type dbLock struct {
	db        *gorm.DB
	lockKey   string
	lockValue string
	tableName string
	mu        sync.Mutex
	unlocked  bool // 标记是否已释放
}

// Unlock 释放锁
func (li *dbLock) Unlock(ctx context.Context) error {
	li.mu.Lock()
	defer li.mu.Unlock()
	if li.unlocked {
		return nil // 幂等性：已经释放的锁再次释放不报错
	}

	// 删除锁记录（只有锁的持有者才能删除）
	result := li.db.WithContext(ctx).Table(li.tableName).
		Where("lock_key = ? AND lock_value = ?", li.lockKey, li.lockValue).
		Delete(&lockModel{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLockNotHeld
	}

	li.unlocked = true
	return nil
}

type dbLocker struct {
	db        *gorm.DB
	tableName string
}

func newDatabaseLocker(db *gorm.DB, table string) *dbLocker {
	if table == "" {
		table = "distributed_lock"
	}

	return &dbLocker{
		db:        db,
		tableName: table,
	}
}

// Acquire 获取锁
func (ml *dbLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
	// 参数校验
	if err := validateKeyAndTTL(key, ttl); err != nil {
		return nil, err
	}

	// 生成 UUID
	value := lockValue()

	// 清理过期的锁
	ml.cleanExpiredLock(ctx, key)

	// 尝试插入锁记录
	expireTime := time.Now().Add(ttl)
	lock := &lockModel{
		LockKey:    key,
		LockValue:  value,
		ExpireTime: expireTime,
	}

	err := ml.db.WithContext(ctx).Table(ml.tableName).Create(lock).Error
	if err != nil {
		// 插入失败，锁已被占用
		return nil, ErrLockAlreadyHeld
	}

	// 创建锁实例
	instance := &dbLock{
		db:        ml.db,
		lockKey:   key,
		lockValue: value,
		tableName: ml.tableName,
	}

	return instance, nil
}

// AcquireWithRetry 带重试获取锁
func (ml *dbLocker) AcquireWithRetry(ctx context.Context, key string, ttl time.Duration, maxRetry int64, interval time.Duration) (Lock, error) {
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

		lock, err := ml.Acquire(ctx, key, ttl)
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

// 清理过期锁
func (ml *dbLocker) cleanExpiredLock(ctx context.Context, key string) error {
	return ml.db.WithContext(ctx).Table(ml.tableName).
		Where("lock_key = ? AND expire_time < ?", key, time.Now()).
		Delete(&lockModel{}).Error
}
