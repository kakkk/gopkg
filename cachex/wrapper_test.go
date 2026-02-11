package cachex

import (
	"context"
	"testing"
	"time"

	"github.com/bytedance/gg/gmap"
	"github.com/bytedance/gg/gptr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestWrapper_Get(t *testing.T) {
	t.Run("l1 hit, l2 miss", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		codec := NewCodecJsonSonic[string]()
		fromL1 := newEntry(gptr.Of("from_l1"), time.Minute)
		l1 := NewMockCacher(ctrl)
		l1.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mustSerialize(t, codec, fromL1), nil).Times(1)
		l2 := NewMockCacher(ctrl)
		w := newWrapper[string](l1, l2, time.Minute, NewCodecJsonSonic[string](), newDefaultLogger())
		got := w.Get(context.Background(), "test")
		assert.Equal(t, gptr.Of("from_l1"), mustGetValue(t, codec, got))
	})
	t.Run("l1 expired, l2 miss", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		codec := NewCodecJsonSonic[string]()
		fromL1 := newEntry(gptr.Of("from_l1"), time.Second)
		time.Sleep(2 * time.Second)
		l1 := NewMockCacher(ctrl)
		l1.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mustSerialize(t, codec, fromL1), nil).Times(1)
		l2 := NewMockCacher(ctrl)
		l2.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		w := newWrapper[string](l1, l2, time.Minute, NewCodecJsonSonic[string](), newDefaultLogger())
		got := w.Get(context.Background(), "test")
		assert.Equal(t, gptr.Of("from_l1"), mustGetValue(t, codec, got))
		assert.True(t, got.IsExpired())
	})
	t.Run("l1 miss, l2 hit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		codec := NewCodecJsonSonic[string]()
		l1 := NewMockCacher(ctrl)
		l1.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		l1.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		fromL2 := newEntry(gptr.Of("from_l2"), time.Minute)
		l2 := NewMockCacher(ctrl)
		l2.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mustSerialize(t, codec, fromL2), nil).Times(1)
		w := newWrapper[string](l1, l2, time.Minute, NewCodecJsonSonic[string](), newDefaultLogger())
		got := w.Get(context.Background(), "test")
		assert.Equal(t, gptr.Of("from_l2"), mustGetValue(t, codec, got))
	})
	t.Run("l1 miss, l2 expired", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		codec := NewCodecJsonSonic[string]()
		l1 := NewMockCacher(ctrl)
		l1.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		fromL2 := newEntry(gptr.Of("from_l2"), time.Second)
		time.Sleep(2 * time.Second)
		l2 := NewMockCacher(ctrl)
		l2.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mustSerialize(t, codec, fromL2), nil).Times(1)
		w := newWrapper[string](l1, l2, time.Minute, NewCodecJsonSonic[string](), newDefaultLogger())
		got := w.Get(context.Background(), "test")
		assert.Equal(t, gptr.Of("from_l2"), mustGetValue(t, codec, got))
		assert.True(t, got.IsExpired())
	})
	t.Run("l1 expired,l2 expired", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		codec := NewCodecJsonSonic[string]()
		fromL1 := newEntry(gptr.Of("from_l1"), time.Second)
		time.Sleep(2 * time.Second)
		l1 := NewMockCacher(ctrl)
		l1.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mustSerialize(t, codec, fromL1), nil).Times(1)
		fromL2 := newEntry(gptr.Of("from_l2"), time.Second)
		time.Sleep(2 * time.Second)
		l2 := NewMockCacher(ctrl)
		l2.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mustSerialize(t, codec, fromL2), nil).Times(1)
		w := newWrapper[string](l1, l2, time.Minute, NewCodecJsonSonic[string](), newDefaultLogger())
		got := w.Get(context.Background(), "test")
		assert.Equal(t, gptr.Of("from_l2"), mustGetValue(t, codec, got))
		assert.True(t, got.IsExpired())
	})
	t.Run("all miss", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		l1 := NewMockCacher(ctrl)
		l1.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		l2 := NewMockCacher(ctrl)
		l2.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		w := newWrapper[string](l1, l2, time.Minute, NewCodecJsonSonic[string](), newDefaultLogger())
		got := w.Get(context.Background(), "test")
		assert.Nil(t, got)
	})
}

func TestWrapper_MGet(t *testing.T) {
	t.Run("l1 hit and expired, l2 hit all, hit all", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		codec := NewCodecJsonSonic[string]()
		l1Hit := newEntry(gptr.Of("from_l1"), time.Minute)
		l1Expired := newEntry(gptr.Of("from_l1"), time.Millisecond)
		l2Hit := newEntry(gptr.Of("from_l2"), time.Minute)
		time.Sleep(time.Second)
		l1 := NewMockCacher(ctrl)
		l1.EXPECT().MGet(gomock.Any(), gomock.Any()).Return(map[string][]byte{
			"l1_hit":     mustSerialize(t, codec, l1Hit),
			"l1_expired": mustSerialize(t, codec, l1Expired),
		}, nil).Times(1)
		l1.EXPECT().MSet(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		l2 := NewMockCacher(ctrl)
		l2.EXPECT().MGet(gomock.Any(), gomock.Any()).Return(map[string][]byte{
			"l1_expired": mustSerialize(t, codec, l2Hit),
			"l2_hit_1":   mustSerialize(t, codec, l2Hit),
			"l2_hit_2":   mustSerialize(t, codec, l2Hit),
		}, nil).Times(1)
		w := newWrapper[string](l1, l2, 2*time.Minute, NewCodecJsonSonic[string](), newDefaultLogger())
		keys := []string{"l1_hit", "l1_expired", "l2_hit_1", "l2_hit_2"}
		fromCache := w.MGet(context.Background(), keys)
		got := gmap.MapValues(fromCache, func(v *entry[string]) *string {
			return mustGetValue(t, codec, v)
		})
		want := map[string]*string{
			"l1_hit":     mustGetValue(t, codec, l1Hit),
			"l1_expired": mustGetValue(t, codec, l2Hit),
			"l2_hit_1":   mustGetValue(t, codec, l2Hit),
			"l2_hit_2":   mustGetValue(t, codec, l2Hit),
		}
		assert.EqualValues(t, want, got)
	})
	t.Run("l1 miss all, l2 hit but some expired, some miss", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		codec := NewCodecJsonSonic[string]()
		l2Hit := newEntry(gptr.Of("from_l2"), time.Minute)
		l2Expired := newEntry(gptr.Of("from_l2"), time.Millisecond)
		time.Sleep(time.Second)
		l1 := NewMockCacher(ctrl)
		l1.EXPECT().MGet(gomock.Any(), gomock.Any()).Return(make(map[string][]byte), nil).Times(1)
		l1.EXPECT().MSet(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		l2 := NewMockCacher(ctrl)
		l2.EXPECT().MGet(gomock.Any(), gomock.Any()).Return(map[string][]byte{
			"hit":     mustSerialize(t, codec, l2Hit),
			"expired": mustSerialize(t, codec, l2Expired),
			"miss":    nil,
		}, nil).Times(1)
		w := newWrapper[string](l1, l2, 2*time.Minute, NewCodecJsonSonic[string](), newDefaultLogger())
		keys := []string{"hit", "expired", "miss"}
		fromCache := w.MGet(context.Background(), keys)
		got := gmap.MapValues(fromCache, func(v *entry[string]) *string {
			return mustGetValue(t, codec, v)
		})
		want := map[string]*string{
			"hit":     mustGetValue(t, codec, l2Hit),
			"expired": mustGetValue(t, codec, l2Expired),
			"miss":    nil,
		}
		assert.EqualValues(t, want, got)
	})
}

func mustSerialize[V any](t *testing.T, codec Codec[V], entry *entry[V]) []byte {
	bytes, err := entry.Serialize(codec)
	assert.NoError(t, err)
	return bytes
}

func mustGetValue[V any](t *testing.T, codec Codec[V], entry *entry[V]) *V {
	val, err := entry.Value(codec)
	assert.NoError(t, err)
	return val
}
