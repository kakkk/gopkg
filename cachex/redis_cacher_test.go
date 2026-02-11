package cachex

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisCacher_SetGet(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cli := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	cacher := NewRedisCacher(cli)
	ctx := context.Background()

	key := "testKeyRedis"
	value := []byte("testValueRedis")
	ttl := time.Second * 5

	// Test Set
	err := cacher.Set(ctx, key, value, ttl)
	assert.NoError(t, err)

	// Test Get - hit
	got, err := cacher.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, got)

	// Test Get - miss after expiry
	s.FastForward(ttl + time.Second) // Simulate time passing
	got, err = cacher.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, got) // Should be nil as it expired

}

func TestRedisCacher_MSetMGet(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cli := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	cacher := NewRedisCacher(cli)
	ctx := context.Background()

	kvs := map[string][]byte{
		"key1Redis": []byte("value1Redis"),
		"key2Redis": []byte("value2Redis"),
		"key3Redis": []byte("value3Redis"),
	}
	ttl := time.Second * 5

	// Test MSet
	err := cacher.MSet(ctx, kvs, ttl)
	assert.NoError(t, err)

	// Test MGet - all hit
	keys := []string{"key1Redis", "key2Redis", "key3Redis", "nonExistentKeyRedis"}
	results, err := cacher.MGet(ctx, keys)
	assert.NoError(t, err)
	assert.Equal(t, kvs["key1Redis"], results["key1Redis"])
	assert.Equal(t, kvs["key2Redis"], results["key2Redis"])
	assert.Equal(t, kvs["key3Redis"], results["key3Redis"])
	assert.Nil(t, results["nonExistentKeyRedis"]) // Non-existent key should return nil

	// Test MGet - after expiry
	s.FastForward(ttl + time.Second) // Simulate time passing
	results, err = cacher.MGet(ctx, keys)
	assert.NoError(t, err)
	assert.Nil(t, results["key1Redis"])
	assert.Nil(t, results["key2Redis"])
	assert.Nil(t, results["key3Redis"])
}

func TestRedisCacher_DeleteMDelete(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cli := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	cacher := NewRedisCacher(cli)
	ctx := context.Background()

	key1 := "deleteKey1Redis"
	key2 := "deleteKey2Redis"
	value := []byte("deleteValueRedis")
	ttl := time.Second * 5

	_ = cacher.Set(ctx, key1, value, ttl)
	_ = cacher.Set(ctx, key2, value, ttl)

	// Test Delete
	err := cacher.Delete(ctx, key1)
	assert.NoError(t, err)
	got, _ := cacher.Get(ctx, key1)
	assert.Nil(t, got) // Should be deleted

	// Test MDelete
	keysToDelete := []string{key2, "nonExistentKeyRedis"}
	err = cacher.MDelete(ctx, keysToDelete)
	assert.NoError(t, err)
	got, _ = cacher.Get(ctx, key2)
	assert.Nil(t, got) // Should be deleted
}

func TestRedisCacher_NewRedisCacher(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cli := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	cacher := NewRedisCacher(cli)
	assert.NotNil(t, cacher)
}
