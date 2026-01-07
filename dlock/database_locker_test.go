package dlock

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDBLock 使用SQLite内存数据库测试分布式锁
func TestDBLock(t *testing.T) {
	// 创建内存SQLite数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NotNil(t, db)

	// 创建表
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS distributed_lock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			lock_key TEXT NOT NULL UNIQUE,
			lock_value TEXT NOT NULL,
			expire_time DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_expire_time ON distributed_lock (expire_time);
	`).Error
	require.NoError(t, err)

	// 设置测试上下文
	ctx := context.Background()

	t.Run("TestAcquireSuccess", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")
		lock, err := locker.Acquire(ctx, "test-key", 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock)

		// 验证锁确实插入到了数据库
		var count int64
		err = db.Table("distributed_lock").Where("lock_key = ?", "test-key").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// 清理
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 验证锁已删除
		err = db.Table("distributed_lock").Where("lock_key = ?", "test-key").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("TestAcquireLockAlreadyHeld", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 第一次获取锁
		lock1, err := locker.Acquire(ctx, "test-key-2", 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock1)

		// 第二次获取同一个锁应该失败
		_, err = locker.Acquire(ctx, "test-key-2", 10*time.Second)
		require.Error(t, err)
		assert.Equal(t, ErrLockAlreadyHeld, err)

		// 清理
		err = lock1.Unlock(ctx)
		require.NoError(t, err)
	})

	t.Run("TestAcquireWithInvalidParams", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 测试空key
		_, err := locker.Acquire(ctx, "", 10*time.Second)
		require.Error(t, err)
		assert.Equal(t, err, ErrInvalidKey)

		// 测试零TTL
		_, err = locker.Acquire(ctx, "test-key", 0)
		require.Error(t, err)
		assert.Equal(t, err, ErrInvalidTTL)

		// 测试负TTL
		_, err = locker.Acquire(ctx, "test-key", -1*time.Second)
		require.Error(t, err)
		assert.Equal(t, err, ErrInvalidTTL)
	})

	t.Run("TestUnlockSuccess", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		lock, err := locker.Acquire(ctx, "test-key-3", 10*time.Second)
		require.NoError(t, err)

		// 验证锁存在
		var count int64
		err = db.Table("distributed_lock").Where("lock_key = ?", "test-key-3").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// 解锁
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 验证锁已删除
		err = db.Table("distributed_lock").Where("lock_key = ?", "test-key-3").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("TestUnlockIdempotent", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		lock, err := locker.Acquire(ctx, "test-key-4", 10*time.Second)
		require.NoError(t, err)

		// 第一次解锁
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 第二次解锁应该也不报错（幂等性）
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 第三次解锁（幂等性）
		err = lock.Unlock(ctx)
		require.NoError(t, err)
	})

	t.Run("TestUnlockNotOwnedLock", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 获取锁
		lock, err := locker.Acquire(ctx, "test-key-5", 10*time.Second)
		require.NoError(t, err)

		// 手动修改数据库中的锁值，模拟锁被其他客户端修改
		result := db.Table("distributed_lock").
			Where("lock_key = ?", "test-key-5").
			Update("lock_value", "wrong-value")
		require.NoError(t, result.Error)

		// 尝试解锁应该失败（因为锁值不匹配）
		err = lock.Unlock(ctx)
		require.Error(t, err)

		// 验证锁仍然存在
		var count int64
		err = db.Table("distributed_lock").Where("lock_key = ?", "test-key-5").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// 清理
		db.Table("distributed_lock").Where("lock_key = ?", "test-key-5").Delete(&lockModel{})
	})

	t.Run("TestExpiredLockCleanup", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 插入一个已经过期的锁
		expiredTime := time.Now().Add(-1 * time.Hour)
		expiredLock := &lockModel{
			LockKey:    "expired-key",
			LockValue:  "expired-value",
			ExpireTime: expiredTime,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err := db.Table("distributed_lock").Create(expiredLock).Error
		require.NoError(t, err)

		// 获取同一个key的锁，应该先清理过期锁
		lock, err := locker.Acquire(ctx, "expired-key", 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock)

		// 验证新锁已创建
		var newLock lockModel
		err = db.Table("distributed_lock").
			Where("lock_key = ?", "expired-key").
			First(&newLock).Error
		require.NoError(t, err)
		assert.NotEqual(t, "expired-value", newLock.LockValue)

		// 清理
		lock.Unlock(ctx)
	})

	t.Run("TestAcquireWithRetrySuccess", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 先获取锁
		lock1, err := locker.Acquire(ctx, "test-key-7", 500*time.Millisecond)
		require.NoError(t, err)

		// 在另一个goroutine中稍后释放锁
		go func() {
			time.Sleep(100 * time.Millisecond)
			lock1.Unlock(ctx)
		}()

		// 尝试获取锁，带重试
		start := time.Now()
		lock2, err := locker.AcquireWithRetry(
			ctx,
			"test-key-7",
			10*time.Second,
			5,                   // 最大重试次数
			50*time.Millisecond, // 重试间隔
		)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, lock2)

		// 应该等待了大约100ms（锁被释放的时间）
		assert.True(t, elapsed >= 100*time.Millisecond, "应该等待锁被释放")
		assert.True(t, elapsed < 300*time.Millisecond, "不应该等待太久")

		// 清理
		lock2.Unlock(ctx)
	})

	t.Run("TestAcquireWithRetryExhausted", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 先获取锁，设置较长的TTL
		lock1, err := locker.Acquire(ctx, "test-key-8", 10*time.Second)
		require.NoError(t, err)

		// 尝试获取锁，带重试，但锁不会释放
		start := time.Now()
		_, err = locker.AcquireWithRetry(
			ctx,
			"test-key-8",
			10*time.Second,
			2,                   // 最大重试次数
			50*time.Millisecond, // 重试间隔
		)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.Equal(t, ErrLockNotAcquired, err)

		// 应该等待了大约100ms（2次重试 * 50ms间隔）
		assert.True(t, elapsed >= 100*time.Millisecond, "应该等待了重试间隔")
		assert.True(t, elapsed < 200*time.Millisecond, "不应该等待超过重试次数")

		// 清理
		lock1.Unlock(ctx)
	})

	t.Run("TestAcquireWithRetryZeroMaxRetry", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 先获取锁
		lock1, err := locker.Acquire(ctx, "test-key-9", 10*time.Second)
		require.NoError(t, err)

		// 尝试获取锁，最大重试次数为0（应该只尝试一次）
		_, err = locker.AcquireWithRetry(
			ctx,
			"test-key-9",
			10*time.Second,
			0, // 最大重试次数为0
			50*time.Millisecond,
		)

		require.Error(t, err)
		assert.Equal(t, ErrLockNotAcquired, err)

		// 清理
		lock1.Unlock(ctx)
	})

	t.Run("TestAcquireWithRetryContextCancel", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 先获取锁
		lock1, err := locker.Acquire(ctx, "test-key-10", 10*time.Second)
		require.NoError(t, err)

		// 创建可取消的context
		ctxWithCancel, cancel := context.WithCancel(ctx)

		// 在另一个goroutine中稍后取消context
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		// 尝试获取锁，带重试
		start := time.Now()
		_, err = locker.AcquireWithRetry(
			ctxWithCancel,
			"test-key-10",
			10*time.Second,
			10, // 较大重试次数
			100*time.Millisecond,
		)
		elapsed := time.Since(start)

		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)

		// 应该在大约50ms后取消
		assert.True(t, elapsed >= 50*time.Millisecond, "应该等待到context被取消")
		assert.True(t, elapsed < 150*time.Millisecond, "不应该等待太久")

		// 清理
		lock1.Unlock(ctx)
	})

	t.Run("TestAcquireWithRetryNegativeMaxRetry", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 测试负数的最大重试次数（应该被视为0）
		lock, err := locker.AcquireWithRetry(
			ctx,
			"test-key-11",
			10*time.Second,
			-5, // 负数重试次数
			50*time.Millisecond,
		)

		// 应该成功获取锁（因为没有人持有）
		require.NoError(t, err)
		require.NotNil(t, lock)

		// 清理
		lock.Unlock(ctx)
	})

	t.Run("TestMultipleLocksIndependence", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 同时获取多个不同的锁
		lock1, err := locker.Acquire(ctx, "lock-1", 10*time.Second)
		require.NoError(t, err)

		lock2, err := locker.Acquire(ctx, "lock-2", 10*time.Second)
		require.NoError(t, err)

		lock3, err := locker.Acquire(ctx, "lock-3", 10*time.Second)
		require.NoError(t, err)

		// 验证三个锁都存在
		var count int64
		err = db.Table("distributed_lock").
			Where("lock_key IN ?", []string{"lock-1", "lock-2", "lock-3"}).
			Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)

		// 依次释放
		err = lock1.Unlock(ctx)
		require.NoError(t, err)

		err = lock2.Unlock(ctx)
		require.NoError(t, err)

		err = lock3.Unlock(ctx)
		require.NoError(t, err)

		// 验证所有锁都已释放
		err = db.Table("distributed_lock").
			Where("lock_key IN ?", []string{"lock-1", "lock-2", "lock-3"}).
			Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("TestLockExpiration", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")

		// 获取一个很快过期的锁
		lock, err := locker.Acquire(ctx, "short-ttl-key", 100*time.Millisecond)
		require.NoError(t, err)
		require.NotNil(t, lock)

		// 等待锁过期
		time.Sleep(200 * time.Millisecond)

		// 再次获取同一个锁应该成功（因为已经过期）
		newLock, err := locker.Acquire(ctx, "short-ttl-key", 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, newLock)

		// 清理
		newLock.Unlock(ctx)
	})

	t.Run("TestCustomTableName", func(t *testing.T) {
		// 创建自定义表
		customTableName := "custom_distributed_lock"
		err := db.Exec(fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				lock_key TEXT NOT NULL UNIQUE,
				lock_value TEXT NOT NULL,
				expire_time DATETIME NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
		`, customTableName)).Error
		require.NoError(t, err)

		// 使用自定义表名的locker
		locker := newDatabaseLocker(db, customTableName)
		lock, err := locker.Acquire(ctx, "custom-table-key", 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock)

		// 验证锁插入到了自定义表
		var count int64
		err = db.Table(customTableName).Where("lock_key = ?", "custom-table-key").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// 清理
		lock.Unlock(ctx)
	})

	t.Run("TestConcurrentLockAcquisition", func(t *testing.T) {
		locker := newDatabaseLocker(db, "distributed_lock")
		key := "concurrent-key"

		// 并发获取锁
		successCh := make(chan Lock, 10)
		errorCh := make(chan error, 10)
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				lock, err := locker.Acquire(ctx, key, 10*time.Second)
				if err != nil {
					errorCh <- err
				} else {
					successCh <- lock
				}
			}(i)
		}

		// 等待所有goroutine完成
		wg.Wait()
		close(successCh)
		close(errorCh)

		// 应该只有一个成功，其他都失败
		var successCount int
		var errorCount int

		for lock := range successCh {
			successCount++
			lock.Unlock(ctx)
		}

		for range errorCh {
			errorCount++
		}

		assert.Equal(t, 1, successCount, "应该只有一个goroutine能成功获取锁")
		assert.Equal(t, 9, errorCount, "应该有9个goroutine获取锁失败")
	})
}

// TestDBLockEdgeCases 测试边界情况
func TestDBLockEdgeCases(t *testing.T) {
	// 创建内存SQLite数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// 创建表
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS distributed_lock (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			lock_key TEXT NOT NULL UNIQUE,
			lock_value TEXT NOT NULL,
			expire_time DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`).Error
	require.NoError(t, err)

	ctx := context.Background()
	locker := newDatabaseLocker(db, "distributed_lock")

	t.Run("TestVeryShortTTL", func(t *testing.T) {
		// 测试非常短的TTL
		lock, err := locker.Acquire(ctx, "short-ttl-key-2", 1*time.Millisecond)
		require.NoError(t, err)
		require.NotNil(t, lock)

		// 等待锁过期
		time.Sleep(10 * time.Millisecond)

		// 应该可以重新获取
		newLock, err := locker.Acquire(ctx, "short-ttl-key-2", 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, newLock)

		newLock.Unlock(ctx)
	})

	t.Run("TestVeryLongTTL", func(t *testing.T) {
		// 测试非常长的TTL
		lock, err := locker.Acquire(ctx, "long-ttl-key", 24*time.Hour)
		require.NoError(t, err)
		require.NotNil(t, lock)

		// 验证过期时间设置正确
		var lockRecord lockModel
		err = db.Table("distributed_lock").
			Where("lock_key = ?", "long-ttl-key").
			First(&lockRecord).Error
		require.NoError(t, err)

		// 过期时间应该在现在+24小时左右
		expectedExpireTime := time.Now().Add(24 * time.Hour)
		timeDiff := expectedExpireTime.Sub(lockRecord.ExpireTime)
		assert.True(t, timeDiff < 5*time.Second, "过期时间设置应该大致正确")

		lock.Unlock(ctx)
	})

	t.Run("TestLockKeyMaxLength", func(t *testing.T) {
		// 测试非常长的锁key
		longKey := ""
		for i := 0; i < 1000; i++ {
			longKey += "a"
		}

		lock, err := locker.Acquire(ctx, longKey, 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock)

		lock.Unlock(ctx)
	})

	t.Run("TestSameKeyDifferentLockers", func(t *testing.T) {
		// 测试多个locker实例操作同一个key
		locker1 := newDatabaseLocker(db, "distributed_lock")
		locker2 := newDatabaseLocker(db, "distributed_lock")

		// locker1获取锁
		lock1, err := locker1.Acquire(ctx, "shared-key", 10*time.Second)
		require.NoError(t, err)

		// locker2尝试获取同一个锁应该失败
		_, err = locker2.Acquire(ctx, "shared-key", 10*time.Second)
		require.Error(t, err)
		assert.Equal(t, ErrLockAlreadyHeld, err)

		// locker1释放锁
		err = lock1.Unlock(ctx)
		require.NoError(t, err)

		// locker2现在应该可以获取锁
		lock2, err := locker2.Acquire(ctx, "shared-key", 10*time.Second)
		require.NoError(t, err)

		lock2.Unlock(ctx)
	})
}
