package cachex

import (
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

type builder[K any, V any] struct {
	namespace string              // 命名空间，用于区分key
	codec     Codec[V]            // 编解码
	expireTTL time.Duration       // 缓存过期时间
	delTTL    time.Duration       // 缓存删除时间
	logger    Logger              // logger
	l1        Cacher              // 一级缓存
	l2        Cacher              // 二级缓存
	genKeyFn  GenKeyFn[K]         // 生成缓存key函数
	loaderFn  LoaderFn[K, V]      // 单个回源函数
	mLoaderFn MultiLoaderFn[K, V] // 批量回源函数
	cacheNil  bool                // 是否缓存空值
	ss        SourceStrategy      // 缓存策略
}

func newBuilder[K any, V any]() CacheBuilder[K, V] {
	return &builder[K, V]{
		namespace: "default",
		codec:     NewCodecJsonSonic[V](),
		ss:        SourceStrategyCacheFirst,
		logger:    newDefaultLogger(),
	}
}

func (b *builder[K, V]) WithNamespace(namespace string) CacheBuilder[K, V] {
	bb := b.copy()
	bb.namespace = namespace
	return bb
}

func (b *builder[K, V]) WithExpireTTL(ttl time.Duration) CacheBuilder[K, V] {
	bb := b.copy()
	bb.expireTTL = ttl
	return bb
}

func (b *builder[K, V]) WithDelTTL(ttl time.Duration) CacheBuilder[K, V] {
	bb := b.copy()
	bb.delTTL = ttl
	return bb
}

func (b *builder[K, V]) WithLogger(logger Logger) CacheBuilder[K, V] {
	bb := b.copy()
	bb.logger = logger
	return bb
}

func (b *builder[K, V]) WithL1(cacher Cacher) CacheBuilder[K, V] {
	bb := b.copy()
	bb.l1 = cacher
	return bb
}

func (b *builder[K, V]) WithL2(cacher Cacher) CacheBuilder[K, V] {
	bb := b.copy()
	bb.l2 = cacher
	return bb
}

func (b *builder[K, V]) WithGenKeyFn(fn GenKeyFn[K]) CacheBuilder[K, V] {
	bb := b.copy()
	bb.genKeyFn = fn
	return bb
}

func (b *builder[K, V]) WithLoader(fn LoaderFn[K, V]) CacheBuilder[K, V] {
	bb := b.copy()
	bb.loaderFn = fn
	return bb
}

func (b *builder[K, V]) WithMultiLoader(fn MultiLoaderFn[K, V]) CacheBuilder[K, V] {
	bb := b.copy()
	bb.mLoaderFn = fn
	return bb
}

func (b *builder[K, V]) WithSourceStrategy(ss SourceStrategy) CacheBuilder[K, V] {
	bb := b.copy()
	bb.ss = ss
	return bb
}

func (b *builder[K, V]) WithCacheNil(cacheNil bool) CacheBuilder[K, V] {
	bb := b.copy()
	bb.cacheNil = cacheNil
	return bb
}

func (b *builder[K, V]) WithCodec(codec Codec[V]) CacheBuilder[K, V] {
	bb := b.copy()
	bb.codec = codec
	return bb
}

func (b *builder[K, V]) Build() (CacheX[K, V], error) {
	bb := b.copy()
	// 命名空间不能为空
	if bb.namespace == "" {
		return nil, fmt.Errorf("namespace not set")
	}
	// 必须要有生成key函数
	if bb.genKeyFn == nil {
		return nil, fmt.Errorf("gen cacheKey fn not set")
	}
	// L2只有在L1不为空时才可以使用
	if bb.l2 != nil && bb.l1 == nil {
		return nil, fmt.Errorf("l1 cacher not set")
	}
	// l1 l2 loader mLoader 都为空
	if bb.loaderFn == nil && bb.mLoaderFn == nil && b.l1 == nil && b.l2 == nil {
		return nil, fmt.Errorf("cacher and loader not set")
	}

	cx := &cachex[K, V]{
		namespace: bb.namespace,
		codec:     bb.codec,
		expireTTL: bb.expireTTL,
		logger:    bb.logger,
		cache:     newWrapper[V](bb.l1, bb.l2, bb.delTTL, bb.codec, bb.logger),
		genKeyFn:  bb.genKeyFn,
		loaderFn:  bb.loaderFn,
		mLoaderFn: bb.mLoaderFn,
		cacheNil:  bb.cacheNil,
		group:     singleflight.Group{},
		mGroup:    singleflight.Group{},
		ss:        bb.ss,
	}
	return cx, nil
}

func (b *builder[K, V]) copy() *builder[K, V] {
	return &builder[K, V]{
		namespace: b.namespace,
		codec:     b.codec,
		expireTTL: b.expireTTL,
		delTTL:    b.delTTL,
		logger:    b.logger,
		l1:        b.l1,
		l2:        b.l2,
		genKeyFn:  b.genKeyFn,
		loaderFn:  b.loaderFn,
		mLoaderFn: b.mLoaderFn,
		cacheNil:  b.cacheNil,
		ss:        b.ss,
	}
}
