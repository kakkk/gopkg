package cachex

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/gg/gmap"
	"github.com/bytedance/gg/gslice"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

type cachex[K any, V any] struct {
	namespace string              // 命名空间，用于区分key
	codec     Codec[V]            // 编解码
	expireTTL time.Duration       // 缓存过期时间
	logger    Logger              // logger
	cache     *wrapper[V]         // 缓存
	genKeyFn  GenKeyFn[K]         // 生成缓存key函数
	loaderFn  LoaderFn[K, V]      // 单个回源函数
	mLoaderFn MultiLoaderFn[K, V] // 批量回源函数
	cacheNil  bool                // 是否缓存空值
	group     singleflight.Group  // 单个回源singleflight
	mGroup    singleflight.Group  // 批量回源singleflight
	ss        SourceStrategy      // 缓存策略
}

func (c *cachex[K, V]) WithSourceStrategy(ss SourceStrategy) CacheX[K, V] {
	cc := c.clone()
	cc.ss = ss
	return cc
}

func (c *cachex[K, V]) Get(ctx context.Context, key K) (*V, error) {
	switch c.ss {
	case SourceStrategyCacheFirst:
		return c.ssCacheFirstGet(ctx, key)
	case SourceStrategySourceFirst:
		return c.ssSourceFirstGet(ctx, key)
	case SourceStrategyCacheOnly:
		return c.ssCacheOnlyGet(ctx, key)
	case SourceStrategySourceOnly:
		return c.ssSourceOnlyGet(ctx, key)
	case SourceStrategyExpiredBackup:
		return c.ssExpiredBackupGet(ctx, key)
	default:
		return nil, fmt.Errorf("invalid source strategy: %v", c.ss)
	}
}

func (c *cachex[K, V]) ssCacheFirstGet(ctx context.Context, key K) (*V, error) {
	// 先读缓存
	cacheKey := c.key(key)
	fromCache := c.cache.Get(ctx, cacheKey)
	// 存在且没过期
	if fromCache != nil && !fromCache.IsExpired() {
		return fromCache.Value(c.codec)
	}
	// 回源
	fromSource, err := c.load(ctx, key)
	if err != nil {
		return nil, err
	}
	// 设置缓存
	_ = c.set(ctx, cacheKey, fromSource)
	return fromSource.Value(c.codec)
}

func (c *cachex[K, V]) ssSourceFirstGet(ctx context.Context, key K) (*V, error) {
	// 回源
	cacheKey := c.key(key)
	fromSource, err := c.load(ctx, key)
	if err != nil {
		// 回源失败，缓存兜底
		fromCache := c.cache.Get(ctx, cacheKey)
		if fromCache != nil && !fromCache.IsExpired() {
			// 有缓存兜底
			return fromCache.Value(c.codec)
		}
		// 没有缓存兜底，返回error
		return nil, err
	}
	// 刷新缓存
	_ = c.set(ctx, cacheKey, fromSource)
	return fromSource.Value(c.codec)
}

func (c *cachex[K, V]) ssCacheOnlyGet(ctx context.Context, key K) (*V, error) {
	cacheKey := c.key(key)
	fromCache := c.cache.Get(ctx, cacheKey)
	// 存在且没过期
	if fromCache != nil && !fromCache.IsExpired() {
		return fromCache.Value(c.codec)
	}
	return nil, nil
}

func (c *cachex[K, V]) ssSourceOnlyGet(ctx context.Context, key K) (*V, error) {
	fromSource, err := c.load(ctx, key)
	if err != nil {
		return nil, err
	}
	return fromSource.Value(c.codec)
}

func (c *cachex[K, V]) ssExpiredBackupGet(ctx context.Context, key K) (*V, error) {
	// 先读缓存
	cacheKey := c.key(key)
	fromCache := c.cache.Get(ctx, cacheKey)
	// 存在且没过期
	if fromCache != nil && !fromCache.IsExpired() {
		return fromCache.Value(c.codec)
	}
	// 回源
	fromSource, err := c.load(ctx, key)
	if err != nil {
		// 回源失败，过期缓存兜底
		if fromCache != nil {
			return fromCache.Value(c.codec)
		}
		// 没有缓存兜底，返回error
		return nil, err
	}
	// 更新缓存
	_ = c.set(ctx, cacheKey, fromSource)
	return fromSource.Value(c.codec)
}

func (c *cachex[K, V]) load(ctx context.Context, key K) (*entry[V], error) {
	if c.loaderFn == nil && c.mLoaderFn == nil {
		return nil, fmt.Errorf("loader not set")
	}
	// 没有配置单个回源函数，从批量回源拿
	if c.loaderFn == nil && c.mLoaderFn != nil {
		vals, err := c.mLoad(ctx, []K{key})
		if err != nil {
			return nil, err
		}
		return vals[c.key(key)], nil
	}
	// 从单个回源拿
	k := c.key(key)
	v, err, _ := c.group.Do(k, func() (interface{}, error) {
		val, err := c.loaderFn(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("loader fn err: %w", err)
		}
		return newEntry(val, c.expireTTL), nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*entry[V]), nil
}

func (c *cachex[K, V]) MGet(ctx context.Context, keys []K) ([]*V, error) {
	switch c.ss {
	case SourceStrategyCacheFirst:
		return c.ssCacheFirstMGet(ctx, keys)
	case SourceStrategySourceFirst:
		return c.ssSourceOnlyMGet(ctx, keys)
	case SourceStrategyCacheOnly:
		return c.ssCacheOnlyMGet(ctx, keys)
	case SourceStrategySourceOnly:
		return c.ssSourceOnlyMGet(ctx, keys)
	case SourceStrategyExpiredBackup:
		return c.ssExpiredBackupMGet(ctx, keys)
	default:
		return nil, fmt.Errorf("invalid source strategy: %v", c.ss)
	}
}

func (c *cachex[K, V]) ssCacheOnlyMGet(ctx context.Context, keys []K) ([]*V, error) {
	cacheKeys := c.keys(keys)
	fromCache := c.cache.MGet(ctx, cacheKeys)
	hit, _, _ := c.groupBatchRes(keys, fromCache)
	return c.packBatchRes(keys, hit), nil
}

func (c *cachex[K, V]) ssSourceOnlyMGet(ctx context.Context, keys []K) ([]*V, error) {
	fromSource, err := c.mLoad(ctx, keys)
	if err != nil {
		return nil, err
	}
	return c.packBatchRes(keys, fromSource), nil
}

func (c *cachex[K, V]) ssCacheFirstMGet(ctx context.Context, keys []K) ([]*V, error) {
	// 读缓存
	fromCache := c.cache.MGet(ctx, c.keys(keys))
	hit, expire, miss := c.groupBatchRes(keys, fromCache)
	if len(miss) == 0 && len(expire) == 0 {
		// 全部命中，直接返回
		return c.packBatchRes(keys, hit), nil
	}
	// 回源
	fromSource, err := c.mLoad(ctx, gslice.Merge(expire, miss))
	if err != nil {
		return nil, err
	}
	_ = c.mSet(ctx, fromSource)
	return c.packBatchRes(keys, gmap.Merge(hit, fromSource)), nil
}

func (c *cachex[K, V]) ssSourceFirstMGet(ctx context.Context, keys []K) ([]*V, error) {
	// 回源
	fromSource, err := c.mLoad(ctx, keys)
	if err != nil {
		// 回源失败，缓存兜底
		fromCache := c.cache.MGet(ctx, c.keys(keys))
		hit, _, _ := c.groupBatchRes(keys, fromCache)
		return c.packBatchRes(keys, hit), nil
	}
	_ = c.mSet(ctx, fromSource)
	return c.packBatchRes(keys, fromSource), nil
}

func (c *cachex[K, V]) ssExpiredBackupMGet(ctx context.Context, keys []K) ([]*V, error) {
	// 读缓存
	fromCache := c.cache.MGet(ctx, c.keys(keys))
	hit, expire, miss := c.groupBatchRes(keys, fromCache)
	if len(miss) == 0 && len(expire) == 0 {
		return c.packBatchRes(keys, hit), nil
	}
	// 回源
	fromSource, err := c.mLoad(ctx, gslice.Merge(expire, miss))
	if err != nil {
		// 回源失败，用缓存数据兜底
		return c.packBatchRes(keys, fromCache), nil
	}
	_ = c.mSet(ctx, fromSource)
	return c.packBatchRes(keys, gmap.Merge(hit, fromSource)), nil
}

func (c *cachex[K, V]) groupBatchRes(keys []K, vals map[string]*entry[V]) (map[string]*entry[V], []K, []K) {
	hit := make(map[string]*entry[V])
	expire := make([]K, 0)
	miss := make([]K, 0)
	for _, key := range keys {
		k := c.key(key)
		v := vals[k]
		if v == nil {
			miss = append(miss, key)
			continue
		}
		if v.IsExpired() {
			expire = append(expire, key)
			continue
		}
		hit[k] = v
	}
	return hit, expire, miss
}

func (c *cachex[K, V]) packBatchRes(keys []K, kvs map[string]*entry[V]) []*V {
	res := make([]*V, len(keys))
	for i, key := range keys {
		entry := kvs[c.key(key)]
		if entry == nil {
			res[i] = nil
			continue
		}
		val, err := entry.Value(c.codec)
		if err != nil {
			res[i] = nil
			continue
		}
		res[i] = val
	}
	return res
}

func (c *cachex[K, V]) mLoad(ctx context.Context, keys []K) (map[string]*entry[V], error) {
	if c.loaderFn == nil && c.mLoaderFn == nil {
		return nil, fmt.Errorf("loader not set")
	}
	// 没有配置批量回源函数，并发从单个回源函数拿
	if c.mLoaderFn == nil && c.loaderFn != nil {
		res := make(map[string]*entry[V])
		mu := sync.Mutex{}
		eg := errgroup.Group{}
		eg.SetLimit(50)
		for _, key := range keys {
			k := key
			eg.Go(func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("loader fn err: panic:%v", r)
					}
				}()
				v, err := c.load(ctx, k)
				if err != nil {
					return err
				}
				mu.Lock()
				defer mu.Unlock()
				res[c.key(k)] = v
				return nil
			})
		}
		err := eg.Wait()
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	// 从批量回源函数拿
	groupKey := "m" + strings.Join(c.keys(keys), ",")
	got, err, _ := c.mGroup.Do(groupKey, func() (interface{}, error) {
		res := make(map[string]*entry[V], len(keys))
		values, err := c.mLoaderFn(ctx, keys)
		if err != nil {
			return nil, fmt.Errorf("mloader fn err: %w", err)
		}
		if len(keys) != len(values) {
			return nil, fmt.Errorf("mloader fn err: len(keys) != len(values)")
		}
		for i, key := range keys {
			res[c.key(key)] = newEntry(values[i], c.expireTTL)
		}
		return res, nil
	})
	if err != nil {
		return nil, err
	}
	return got.(map[string]*entry[V]), nil
}

func (c *cachex[K, V]) Set(ctx context.Context, key K, value *V) error {
	return c.set(ctx, c.key(key), newEntry(value, c.expireTTL))
}

func (c *cachex[K, V]) set(ctx context.Context, key string, val *entry[V]) error {
	if val == nil {
		return nil
	}
	if val.IsNil() && !c.cacheNil {
		return nil
	}
	return c.cache.Set(ctx, key, val)
}

func (c *cachex[K, V]) MSet(ctx context.Context, keys []K, values []*V) error {
	if len(keys) != len(values) {
		return fmt.Errorf("keys values length not equal")
	}
	kvs := make(map[string]*entry[V])
	for i := 0; i < len(keys); i++ {
		kvs[c.key(keys[i])] = newEntry(values[i], c.expireTTL)
	}
	return c.mSet(ctx, kvs)
}

func (c *cachex[K, V]) mSet(ctx context.Context, kvs map[string]*entry[V]) error {
	data := make(map[string]*entry[V])
	for k, v := range kvs {
		if v != nil {
			if v.IsNil() && !c.cacheNil {
				continue
			}
		}
		data[k] = v
	}
	return c.cache.MSet(ctx, kvs)
}

func (c *cachex[K, V]) Del(ctx context.Context, key K) error {
	return c.cache.Delete(ctx, c.key(key))
}

func (c *cachex[K, V]) MDel(ctx context.Context, keys []K) error {
	return c.cache.MDelete(ctx, c.keys(keys))
}

func (c *cachex[K, V]) key(key K) string {
	return c.namespace + ":" + c.genKeyFn(key)
}

func (c *cachex[K, V]) keys(keys []K) []string {
	res := make([]string, len(keys))
	for i := 0; i < len(keys); i++ {
		res[i] = c.key(keys[i])
	}
	return gslice.Uniq(res)
}

func (c *cachex[K, V]) clone() *cachex[K, V] {
	return &cachex[K, V]{
		namespace: c.namespace,
		codec:     c.codec,
		expireTTL: c.expireTTL,
		genKeyFn:  c.genKeyFn,
		loaderFn:  c.loaderFn,
		mLoaderFn: c.mLoaderFn,
		cacheNil:  c.cacheNil,
		ss:        c.ss,
	}
}
