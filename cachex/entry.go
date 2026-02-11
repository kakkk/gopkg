package cachex

import (
	"encoding/binary"
	"fmt"
	"time"
)

// +------------------------+------------------------+--------+----------------+
// | CreateAt               | TTL                    | IsNil  | Value          |
// +------------------------+------------------------+--------+----------------+
// ↑                        ↑                        ↑        ↑
// 第0字节                  第8字节                  第16字节  第17字节
//
// 总长度 = 17字节(固定头部) + len(Value)字节

const (
	bytesCreateAtSize = 8
	bytesTTLSize      = 8
	bytesIsNilSize    = 1
	bytesHeaderSize   = bytesCreateAtSize + bytesTTLSize + bytesIsNilSize
)

type entry[V any] struct {
	createAt int64         // 创建时间
	ttl      time.Duration // 业务过期时间
	valBytes []byte        // value序列化后的值
	val      *V            // 缓存值
	isNil    uint8         // 是否为空值
}

func (e *entry[V]) Serialize(codec Codec[V]) ([]byte, error) {
	if len(e.valBytes) == 0 && e.val != nil {
		bytes, err := codec.Marshal(e.val)
		if err != nil {
			return nil, fmt.Errorf("cachex: failed to marshal value: %v", err)
		}
		e.valBytes = bytes
	}
	totalLen := bytesHeaderSize + len(e.valBytes)
	buffer := make([]byte, totalLen)
	binary.LittleEndian.PutUint64(buffer[0:bytesCreateAtSize], uint64(e.createAt))
	binary.LittleEndian.PutUint64(buffer[bytesCreateAtSize:bytesCreateAtSize+bytesTTLSize], uint64(e.ttl))
	buffer[bytesCreateAtSize+bytesTTLSize] = e.isNil
	copy(buffer[bytesHeaderSize:], e.valBytes)
	return buffer, nil
}

func (e *entry[V]) IsExpired() bool {
	// expire小于等于0，不过期
	if e.ttl <= 0 {
		return false
	}
	// 创建时间+业务过期时间小于当前时间, 已过期
	if e.createAt+e.ttl.Milliseconds() < time.Now().UnixMilli() {
		return true
	}
	return false
}

func (e *entry[V]) IsNil() bool {
	return e.isNil == 1
}

func (e *entry[V]) Value(codec Codec[V]) (*V, error) {
	if e == nil {
		return nil, nil
	}
	if e.IsNil() {
		return nil, nil
	}
	if e.val != nil {
		return e.val, nil
	}
	val, err := codec.Unmarshal(e.valBytes)
	if err != nil {
		panic(fmt.Errorf("cachex: failed to unmarshal value: %v", err))
	}
	return val, nil
}

func (e *entry[V]) CreateAt() int64 {
	return e.createAt
}

func newEntry[V any](val *V, ttl time.Duration) *entry[V] {
	if val == nil {
		return &entry[V]{
			createAt: time.Now().UnixMilli(),
			ttl:      ttl,
			valBytes: nil,
			val:      nil,
			isNil:    1,
		}
	}
	return &entry[V]{
		createAt: time.Now().UnixMilli(),
		ttl:      ttl,
		valBytes: nil,
		val:      val,
		isNil:    0,
	}
}

func deserializeEntry[V any](bytes []byte) *entry[V] {
	if len(bytes) < 17 {
		return nil
	}
	return &entry[V]{
		createAt: int64(binary.LittleEndian.Uint64(bytes[0:bytesCreateAtSize])),
		ttl:      time.Duration(int64(binary.LittleEndian.Uint64(bytes[bytesCreateAtSize : bytesCreateAtSize+bytesTTLSize]))),
		isNil:    bytes[bytesCreateAtSize+bytesTTLSize],
		valBytes: bytes[bytesHeaderSize:],
	}
}
