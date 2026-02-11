package cachex

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type wrapper[V any] struct {
	l1       Cacher
	l2       Cacher
	cacheNil bool
	delTTL   time.Duration
	codec    Codec[V]
	logger   Logger
}

func newWrapper[V any](l1 Cacher, l2 Cacher, delTTL time.Duration, codec Codec[V], logger Logger) *wrapper[V] {
	return &wrapper[V]{
		l1:     l1,
		l2:     l2,
		delTTL: delTTL,
		codec:  codec,
		logger: logger,
	}
}

func (w *wrapper[V]) Get(ctx context.Context, key string) *entry[V] {
	fromL1 := w.get(ctx, w.l1, key)
	if fromL1 != nil && !fromL1.IsExpired() {
		return fromL1
	}
	fromL2 := w.get(ctx, w.l2, key)
	if fromL2 != nil && !fromL2.IsExpired() {
		_ = w.set(ctx, w.l1, key, fromL2, w.getDelTTL(1))
		return fromL2
	}
	return w.latest(fromL1, fromL2)
}

func (w *wrapper[V]) get(ctx context.Context, cacher Cacher, key string) *entry[V] {
	if cacher == nil {
		return nil
	}
	val, err := cacher.Get(ctx, key)
	if err != nil {
		w.logger.Warnf(ctx, "cachex: cacher get error: %v", err)
		return nil
	}
	if val == nil {
		return nil
	}
	return deserializeEntry[V](val)
}

func (w *wrapper[V]) MGet(ctx context.Context, keys []string) map[string]*entry[V] {
	fromL1 := w.mGet(ctx, w.l1, keys)
	miss := make([]string, 0)
	hit := make(map[string]*entry[V])
	for _, key := range keys {
		val := fromL1[key]
		hit[key] = val
		if val == nil || val.IsExpired() {
			miss = append(miss, key)
		}
	}
	if len(miss) == 0 {
		return hit
	}

	fromL2 := w.mGet(ctx, w.l2, miss)
	hitL2 := make(map[string]*entry[V])
	for _, key := range keys {
		val := fromL2[key]
		hit[key] = w.latest(hit[key], val)
		if val != nil && !val.IsExpired() {
			hitL2[key] = val
		}
	}
	_ = w.mSet(ctx, w.l1, hitL2, w.getDelTTL(1))
	return hit
}

func (w *wrapper[V]) mGet(ctx context.Context, cacher Cacher, keys []string) map[string]*entry[V] {
	data := make(map[string]*entry[V])
	if cacher == nil {
		return data
	}
	kvs, err := cacher.MGet(ctx, keys)
	if err != nil {
		w.logger.Warnf(ctx, "cachex: cacher mget error: %v", err)
		return data
	}
	for k, v := range kvs {
		if v == nil {
			continue
		}
		data[k] = deserializeEntry[V](v)
	}
	return data
}

func (w *wrapper[V]) Set(ctx context.Context, key string, val *entry[V]) error {
	l2Err := w.set(ctx, w.l2, key, val, w.getDelTTL(2))
	l1Err := w.set(ctx, w.l1, key, val, w.getDelTTL(1))
	if l1Err != nil || l2Err != nil {
		return fmt.Errorf("cachex: cacher set error, l1:%w, l2:%w", l1Err, l2Err)
	}
	return nil
}

func (w *wrapper[V]) set(ctx context.Context, cacher Cacher, key string, val *entry[V], ttl time.Duration) error {
	if cacher == nil || val == nil {
		return nil
	}
	bytes, err := val.Serialize(w.codec)
	if err != nil {
		return err
	}
	err = cacher.Set(ctx, key, bytes, ttl)
	if err != nil {
		return err
	}
	return nil
}

func (w *wrapper[V]) MSet(ctx context.Context, kvs map[string]*entry[V]) error {
	l2Err := w.mSet(ctx, w.l2, kvs, w.getDelTTL(2))
	l1Err := w.mSet(ctx, w.l1, kvs, w.getDelTTL(1))
	if l1Err != nil || l2Err != nil {
		return fmt.Errorf("cachex: mSet cacher error, l1:%w, l2:%w", l1Err, l2Err)
	}
	return nil
}

func (w *wrapper[V]) mSet(ctx context.Context, cacher Cacher, kvs map[string]*entry[V], ttl time.Duration) error {
	if cacher == nil {
		return nil
	}
	if len(kvs) == 0 {
		return nil
	}
	data := make(map[string][]byte)
	for k, v := range kvs {
		if v == nil {
			continue
		}
		if v.IsNil() && !w.cacheNil {
			continue
		}
		var err error
		data[k], err = v.Serialize(w.codec)
		if err != nil {
			return err
		}
	}
	err := cacher.MSet(ctx, data, ttl)
	if err != nil {
		return err
	}
	return nil
}

func (w *wrapper[V]) Delete(ctx context.Context, key string) error {
	l2Err := w.delete(ctx, w.l2, key)
	l1Err := w.delete(ctx, w.l1, key)
	if l1Err != nil || l2Err != nil {
		return fmt.Errorf("cachex: cahcer delete error: l1:%w, l2:%w", l1Err, l2Err)
	}
	return nil
}

func (w *wrapper[V]) delete(ctx context.Context, cacher Cacher, key string) error {
	if cacher == nil {
		return nil
	}
	err := cacher.Delete(ctx, key)
	if err != nil {
		return err
	}
	return nil
}

func (w *wrapper[V]) MDelete(ctx context.Context, keys []string) error {
	l2Err := w.mDelete(ctx, w.l2, keys)
	l1Err := w.mDelete(ctx, w.l1, keys)
	if l1Err != nil || l2Err != nil {
		return fmt.Errorf("cachex: cahcer mDelete error: l1:%w, l2:%w", l1Err, l2Err)
	}
	return nil
}

func (w *wrapper[V]) mDelete(ctx context.Context, cacher Cacher, keys []string) error {
	if cacher == nil {
		return nil
	}
	if len(keys) == 0 {
		return nil
	}
	err := cacher.MDelete(ctx, keys)
	if err != nil {
		return err
	}
	return nil
}

func (w *wrapper[V]) getDelTTL(level int) time.Duration {
	if level != 1 && level != 2 {
		// never reach here
		panic("cachex: invalid level")
	}
	// 增加随机值防止集中过期
	r := time.Duration(rand.Int63n(1000)) * time.Millisecond
	if level == 1 {
		return w.delTTL + r
	}
	// L2缓存的ttl要略大于L1
	return time.Duration(float64(w.delTTL)*1.3) + r
}

func (w *wrapper[V]) latest(e1, e2 *entry[V]) *entry[V] {
	if e1 == nil && e2 == nil {
		return nil
	}
	if e1 != nil && e2 != nil {
		if e1.CreateAt() > e2.CreateAt() {
			return e1
		}
		return e2
	}
	if e1 != nil {
		return e1
	}
	return e2
}
