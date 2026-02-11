package cachex

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coocood/freecache"
)

// localCache 本地缓存实现
type localCache struct {
	fc *freecache.Cache
}

func NewLocalCacher(sizeMB int) Cacher {
	return &localCache{
		fc: freecache.NewCache(sizeMB),
	}
}

func (l *localCache) Get(_ context.Context, key string) ([]byte, error) {
	val, err := l.fc.Get(stringToBytes(key))
	if err != nil {
		if errors.Is(err, freecache.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("freecache error: %w", err)
	}
	return val, nil
}

func (l *localCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, key := range keys {
		val, err := l.Get(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("freecache error: %w", err)
		}
		result[key] = val
	}
	return result, nil
}

func (l *localCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	err := l.fc.Set(stringToBytes(key), val, int(ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("freecache error: %w", err)
	}
	return nil
}

func (l *localCache) MSet(ctx context.Context, kvs map[string][]byte, ttl time.Duration) error {
	success := make([]string, 0, len(kvs))
	for k, v := range kvs {
		err := l.Set(ctx, k, v, ttl)
		if err != nil {
			_ = l.MDelete(ctx, success)
			return err
		}
		success = append(success, k)
	}
	return nil
}

func (l *localCache) Delete(_ context.Context, key string) error {
	l.fc.Del([]byte(key))
	return nil
}

func (l *localCache) MDelete(_ context.Context, keys []string) error {
	for _, key := range keys {
		l.fc.Del([]byte(key))
	}
	return nil
}
