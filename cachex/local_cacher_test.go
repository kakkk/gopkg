package cachex

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocalCacher_SetGet(t *testing.T) {
	cacher := NewLocalCacher(1) // 1MB cache
	ctx := context.Background()

	key := "testKey"
	value := []byte("testValue")
	ttl := time.Second * 5

	// Test Set
	err := cacher.Set(ctx, key, value, ttl)
	assert.NoError(t, err)

	// Test Get - hit
	got, err := cacher.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, got)

	// Test Get - miss after expiry
	time.Sleep(ttl + time.Second)
	got, err = cacher.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, got) // Should be nil as it expired

}

func TestLocalCacher_MSetMGet(t *testing.T) {
	cacher := NewLocalCacher(1) // 1MB cache
	ctx := context.Background()

	kvs := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}
	ttl := time.Second * 5

	// Test MSet
	err := cacher.MSet(ctx, kvs, ttl)
	assert.NoError(t, err)

	// Test MGet - all hit
	keys := []string{"key1", "key2", "key3", "nonExistentKey"}
	results, err := cacher.MGet(ctx, keys)
	assert.NoError(t, err)
	assert.Equal(t, kvs["key1"], results["key1"])
	assert.Equal(t, kvs["key2"], results["key2"])
	assert.Equal(t, kvs["key3"], results["key3"])
	assert.Nil(t, results["nonExistentKey"]) // Non-existent key should return nil

	// Test MGet - after expiry
	time.Sleep(ttl + time.Second)
	results, err = cacher.MGet(ctx, keys)
	assert.NoError(t, err)
	assert.Nil(t, results["key1"])
	assert.Nil(t, results["key2"])
	assert.Nil(t, results["key3"])
}

func TestLocalCacher_DeleteMDelete(t *testing.T) {
	cacher := NewLocalCacher(1) // 1MB cache
	ctx := context.Background()

	key1 := "deleteKey1"
	key2 := "deleteKey2"
	value := []byte("deleteValue")
	ttl := time.Second * 5

	_ = cacher.Set(ctx, key1, value, ttl)
	_ = cacher.Set(ctx, key2, value, ttl)

	// Test Delete
	err := cacher.Delete(ctx, key1)
	assert.NoError(t, err)
	got, _ := cacher.Get(ctx, key1)
	assert.Nil(t, got) // Should be deleted

	// Test MDelete
	keysToDelete := []string{key2, "nonExistentKey"}
	err = cacher.MDelete(ctx, keysToDelete)
	assert.NoError(t, err)
	got, _ = cacher.Get(ctx, key2)
	assert.Nil(t, got) // Should be deleted
}

func TestLocalCacher_NewLocalCacher(t *testing.T) {
	cacher := NewLocalCacher(1024) // 1KB cache
	assert.NotNil(t, cacher)
	// You can add more specific tests for capacity if freecache exposes it
}
