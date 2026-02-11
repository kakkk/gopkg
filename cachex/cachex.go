package cachex

import (
	"context"
	"time"
)

type SourceStrategy int64

const (
	SourceStrategyCacheFirst    SourceStrategy = 1 // 缓存优先
	SourceStrategySourceFirst   SourceStrategy = 2 // 回源优先
	SourceStrategyCacheOnly     SourceStrategy = 3 // 仅缓存
	SourceStrategySourceOnly    SourceStrategy = 4 // 仅回源
	SourceStrategyExpiredBackup SourceStrategy = 5 // 缓存优先，回源失败使用缓存兜底
)

type LoaderFn[K, V any] func(ctx context.Context, key K) (*V, error)
type MultiLoaderFn[K, V any] func(ctx context.Context, keys []K) ([]*V, error)
type GenKeyFn[K any] func(key K) string

type CacheBuilder[K, V any] interface {
	WithNamespace(namespace string) CacheBuilder[K, V]         // 设置命名空间，用于区分不同缓存
	WithExpireTTL(ttl time.Duration) CacheBuilder[K, V]        // 设置缓存失效时间
	WithDelTTL(ttl time.Duration) CacheBuilder[K, V]           // 缓存删除时间
	WithLogger(logger Logger) CacheBuilder[K, V]               // logger
	WithL1(cacher Cacher) CacheBuilder[K, V]                   // 设置一级缓存
	WithL2(cacher Cacher) CacheBuilder[K, V]                   // 设置二级缓存
	WithGenKeyFn(fn GenKeyFn[K]) CacheBuilder[K, V]            // 设置缓存Key生成函数
	WithLoader(fn LoaderFn[K, V]) CacheBuilder[K, V]           // 设置单个回源
	WithMultiLoader(fn MultiLoaderFn[K, V]) CacheBuilder[K, V] // 设置批量回源
	WithSourceStrategy(ss SourceStrategy) CacheBuilder[K, V]   // 设置回源策略
	WithCacheNil(cacheNil bool) CacheBuilder[K, V]             // 设置是否缓存空值，即回源若不存在，则缓存空值
	WithCodec(codec Codec[V]) CacheBuilder[K, V]               // 编解码
	Build() (CacheX[K, V], error)                              // 创建缓存实例
}

type CacheX[K, V any] interface {
	WithSourceStrategy(ss SourceStrategy) CacheX[K, V]
	Get(ctx context.Context, key K) (*V, error)
	Set(ctx context.Context, key K, value *V) error
	Del(ctx context.Context, key K) error
	MGet(ctx context.Context, keys []K) ([]*V, error)
	MSet(ctx context.Context, keys []K, values []*V) error
	MDel(ctx context.Context, keys []K) error
}

type Cacher interface {
	Get(ctx context.Context, key string) ([]byte, error)
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	MSet(ctx context.Context, kvs map[string][]byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	MDelete(ctx context.Context, keys []string) error
}

type Logger interface {
	Infof(ctx context.Context, format string, v ...interface{})
	Warnf(ctx context.Context, format string, v ...interface{})
	Errorf(ctx context.Context, format string, v ...interface{})
}

func New[K, V any]() CacheBuilder[K, V] {
	return newBuilder[K, V]()
}
