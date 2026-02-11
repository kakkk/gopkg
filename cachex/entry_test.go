package cachex

import (
	"testing"
	"time"

	"github.com/bytedance/gg/gptr"
	"github.com/stretchr/testify/assert"
)

func TestEntry(t *testing.T) {
	t.Run("not nil", func(t *testing.T) {
		ttl := 5 * time.Second
		codec := NewCodecJsonSonic[string]()
		e := newEntry[string](gptr.Of("hello"), ttl)
		assert.False(t, e.IsNil())
		assert.False(t, e.IsExpired())
		val, err := e.Value(codec)
		assert.NoError(t, err)
		assert.EqualValues(t, gptr.Of("hello"), val)
		time.Sleep(10 * time.Second)
		assert.True(t, e.IsExpired())
		bytes, err := e.Serialize(codec)
		assert.NoError(t, err)
		e2 := deserializeEntry[string](bytes)
		val, err = e2.Value(codec)
		assert.NoError(t, err)
		assert.EqualValues(t, gptr.Of("hello"), val)
		assert.Equal(t, e.createAt, e2.createAt)
		assert.Equal(t, e.ttl, e2.ttl)
		assert.Equal(t, e.isNil, e2.isNil)
	})
	t.Run("is nil", func(t *testing.T) {
		ttl := 5 * time.Second
		codec := NewCodecJsonSonic[string]()
		e := newEntry[string](nil, ttl)
		assert.True(t, e.IsNil())
		assert.False(t, e.IsExpired())
		val, err := e.Value(codec)
		assert.NoError(t, err)
		assert.Nil(t, val)
		time.Sleep(10 * time.Second)
		assert.True(t, e.IsExpired())
		bytes, err := e.Serialize(codec)
		assert.NoError(t, err)
		e2 := deserializeEntry[string](bytes)
		val, err = e2.Value(codec)
		assert.NoError(t, err)
		assert.Nil(t, val)
		assert.Equal(t, e.createAt, e2.createAt)
		assert.Equal(t, e.ttl, e2.ttl)
		assert.Equal(t, e.isNil, e2.isNil)
	})
}
