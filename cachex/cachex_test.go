package cachex

import (
	"context"
	"testing"
	"time"

	"github.com/bytedance/gg/gptr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCachex_Get(t *testing.T) {
	t.Run("cache hit, source hit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		l1 := NewMockCacher(ctrl)
		codec := NewCodecJsonSonic[string]()
		fromCache := newEntry(gptr.Of("from_cache"), time.Minute)
		bytes, err := fromCache.Serialize(codec)
		assert.NoError(t, err)
		l1.EXPECT().Get(gomock.Any(), gomock.Any()).Return(bytes, nil).AnyTimes()
		l1.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		loaderFn := func(ctx context.Context, key string) (*string, error) { return gptr.Of("from_source"), nil }
		genKeyFn := func(key string) string { return key }
		b := New[string, string]().
			WithLoader(loaderFn).
			WithL1(l1).
			WithGenKeyFn(genKeyFn).
			WithCodec(NewCodecJsonSonic[string]()).
			WithExpireTTL(time.Minute)
		ctx := context.Background()

		t.Run("SourceStrategyCacheFirst", func(t *testing.T) {
			cx, err := b.WithSourceStrategy(SourceStrategyCacheFirst).Build()
			assert.NoError(t, err)
			assert.NotNil(t, cx)
			got, err := cx.Get(ctx, "test")
			assert.NoError(t, err)
			assert.EqualValues(t, gptr.Of("from_cache"), got)
		})
		t.Run("SourceStrategySourceFirst", func(t *testing.T) {
			cx, err := b.WithSourceStrategy(SourceStrategySourceFirst).Build()
			assert.NoError(t, err)
			assert.NotNil(t, cx)
			got, err := cx.Get(ctx, "test")
			assert.NoError(t, err)
			assert.EqualValues(t, gptr.Of("from_source"), got)
		})
		t.Run("SourceStrategyCacheOnly", func(t *testing.T) {
			cx, err := b.WithSourceStrategy(SourceStrategyCacheOnly).Build()
			assert.NoError(t, err)
			assert.NotNil(t, cx)
			got, err := cx.Get(ctx, "test")
			assert.NoError(t, err)
			assert.EqualValues(t, gptr.Of("from_cache"), got)
		})
		t.Run("SourceStrategySourceOnly", func(t *testing.T) {
			cx, err := b.WithSourceStrategy(SourceStrategySourceOnly).Build()
			assert.NoError(t, err)
			assert.NotNil(t, cx)
			got, err := cx.Get(ctx, "test")
			assert.NoError(t, err)
			assert.EqualValues(t, gptr.Of("from_source"), got)
		})
		t.Run("SourceStrategyExpiredBackup", func(t *testing.T) {
			cx, err := b.WithSourceStrategy(SourceStrategyExpiredBackup).Build()
			assert.NoError(t, err)
			assert.NotNil(t, cx)
			got, err := cx.Get(ctx, "test")
			assert.NoError(t, err)
			assert.EqualValues(t, gptr.Of("from_cache"), got)
		})
	})
}
