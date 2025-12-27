package dlock

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLocker(t *testing.T) {
	// 创建miniredis实例
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	ctx := context.Background()

	t.Run("TestAcquireSuccess", func(t *testing.T) {
		locker := newRedisLocker(client)
		lock, err := locker.Acquire(ctx, "test-key", 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock)

		// 验证锁确实设置到了Redis
		val, err := client.Get(ctx, "test-key").Result()
		require.NoError(t, err)
		assert.NotEmpty(t, val)

		// 清理
		err = lock.Unlock(ctx)
		require.NoError(t, err)
	})

	t.Run("TestAcquireLockAlreadyHeld", func(t *testing.T) {
		locker := newRedisLocker(client)

		// 第一次获取锁
		lock1, err := locker.Acquire(ctx, "test-key-2", 10*time.Second)
		require.NoError(t, err)

		// 第二次获取同一个锁应该失败
		_, err = locker.Acquire(ctx, "test-key-2", 10*time.Second)
		require.Error(t, err)
		assert.Equal(t, ErrLockAlreadyHeld, err)

		// 清理
		err = lock1.Unlock(ctx)
		require.NoError(t, err)
	})

	t.Run("TestAcquireWithInvalidParams", func(t *testing.T) {
		locker := newRedisLocker(client)

		// 测试空key
		_, err := locker.Acquire(ctx, "", 10*time.Second)
		require.Error(t, err)

		// 测试零TTL
		_, err = locker.Acquire(ctx, "test-key", 0)
		require.Error(t, err)

		// 测试负TTL
		_, err = locker.Acquire(ctx, "test-key", -1*time.Second)
		require.Error(t, err)
	})

	t.Run("TestUnlockSuccess", func(t *testing.T) {
		locker := newRedisLocker(client)

		lock, err := locker.Acquire(ctx, "test-key-3", 10*time.Second)
		require.NoError(t, err)

		// 验证锁存在
		exists, err := client.Exists(ctx, "test-key-3").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		// 解锁
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 验证锁已删除
		exists, err = client.Exists(ctx, "test-key-3").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), exists)
	})

	t.Run("TestUnlockIdempotent", func(t *testing.T) {
		locker := newRedisLocker(client)

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
		locker := newRedisLocker(client)

		// 获取锁
		lock, err := locker.Acquire(ctx, "test-key-5", 10*time.Second)
		require.NoError(t, err)

		// 手动修改Redis中的锁值，模拟锁被其他客户端修改
		err = client.Set(ctx, "test-key-5", "wrong-value", 10*time.Second).Err()
		require.NoError(t, err)

		// 尝试解锁应该失败
		err = lock.Unlock(ctx)
		require.Error(t, err)
		assert.Equal(t, ErrLockNotHeld, err)

		// 清理
		client.Del(ctx, "test-key-5")
	})

	t.Run("TestUnlockExpiredLock", func(t *testing.T) {
		locker := newRedisLocker(client)

		// 获取一个很快过期的锁
		lock, err := locker.Acquire(ctx, "test-key-6", 100*time.Millisecond)
		require.NoError(t, err)

		// 等待锁过期
		time.Sleep(200 * time.Millisecond)

		// 解锁过期的锁应该不报错
		err = lock.Unlock(ctx)
		require.NoError(t, err)
	})

	t.Run("TestAcquireWithRetrySuccess", func(t *testing.T) {
		locker := newRedisLocker(client)

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
		locker := newRedisLocker(client)

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
		locker := newRedisLocker(client)

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
		locker := newRedisLocker(client)

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
		locker := newRedisLocker(client)

		// 测试负数的最大重试次数（应该被视为0）
		_, err := locker.AcquireWithRetry(
			ctx,
			"test-key-11",
			10*time.Second,
			-5, // 负数重试次数
			50*time.Millisecond,
		)

		// 应该成功获取锁（因为没有人持有）
		require.NoError(t, err)

		// 清理 - 需要先获取锁对象
		lock, err := locker.Acquire(ctx, "test-key-11", 10*time.Second)
		if err == nil {
			lock.Unlock(ctx)
		}
	})

	t.Run("TestMultipleLocksIndependence", func(t *testing.T) {
		locker := newRedisLocker(client)

		// 同时获取多个不同的锁
		lock1, err := locker.Acquire(ctx, "lock-1", 10*time.Second)
		require.NoError(t, err)

		lock2, err := locker.Acquire(ctx, "lock-2", 10*time.Second)
		require.NoError(t, err)

		lock3, err := locker.Acquire(ctx, "lock-3", 10*time.Second)
		require.NoError(t, err)

		// 验证三个锁都存在
		count, err := client.Exists(ctx, "lock-1", "lock-2", "lock-3").Result()
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
		count, err = client.Exists(ctx, "lock-1", "lock-2", "lock-3").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("TestLuaScriptAtomicity", func(t *testing.T) {
		locker := newRedisLocker(client)

		// 获取锁
		lock, err := locker.Acquire(ctx, "atomic-key", 10*time.Second)
		require.NoError(t, err)

		// 获取锁的值
		originalValue, err := client.Get(ctx, "atomic-key").Result()
		require.NoError(t, err)

		// 直接修改锁的值（模拟其他客户端修改）
		err = client.Set(ctx, "atomic-key", "hijacked-value", 10*time.Second).Err()
		require.NoError(t, err)

		// 验证锁值已被修改
		newValue, err := client.Get(ctx, "atomic-key").Result()
		require.NoError(t, err)
		assert.Equal(t, "hijacked-value", newValue)
		assert.NotEqual(t, originalValue, newValue)

		// 执行解锁 - 应该失败，因为锁值不匹配
		err = lock.Unlock(ctx)
		require.Error(t, err)
		assert.Equal(t, ErrLockNotHeld, err)

		// 验证锁仍然存在（因为解锁失败）
		exists, err := client.Exists(ctx, "atomic-key").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		// 清理
		client.Del(ctx, "atomic-key")
	})
}

func TestRedisLockConcurrent(t *testing.T) {
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	ctx := context.Background()
	locker := newRedisLocker(client)
	key := "concurrent-key"

	// 并发测试
	const goroutines = 10
	var successCount int32
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			lock, err := locker.Acquire(ctx, key, 100*time.Millisecond)
			if err != nil {
				// 获取锁失败是正常的，因为并发
				return
			}

			// 成功获取锁
			atomic.AddInt32(&successCount, 1)

			// 持有锁一小段时间
			time.Sleep(20 * time.Millisecond)

			// 释放锁
			lock.Unlock(ctx)
		}(i)
	}

	wg.Wait()

	// 在并发情况下，应该只有一个goroutine能成功获取锁
	assert.Equal(t, int32(1), atomic.LoadInt32(&successCount))
}
