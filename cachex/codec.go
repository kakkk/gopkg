package cachex

import (
	"encoding/json"
	"errors"

	"github.com/bytedance/sonic"
)

type Codec[V any] interface {
	Marshal(v *V) ([]byte, error)
	Unmarshal(data []byte) (*V, error)
}

func NewCodecBytesDirect() Codec[[]byte] {
	return &bytesDirect[[]byte]{}
}

type bytesDirect[V []byte] struct{}

func (bytesDirect[V]) Marshal(v *V) ([]byte, error) {
	if v == nil {
		return nil, errors.New("value is nil")
	}
	return *v, nil
}

func (bytesDirect[V]) Unmarshal(data []byte) (*V, error) {
	v := V(data)
	return &v, nil
}

func NewCodecRawString() Codec[string] {
	return &rawString[string]{}
}

type rawString[V string] struct{}

func (rawString[V]) Marshal(v *V) ([]byte, error) {
	if v == nil {
		return nil, errors.New("value is nil")
	}
	return []byte(*v), nil
}

func (rawString[V]) Unmarshal(data []byte) (*V, error) {
	v := V(data)
	return &v, nil
}

// NewCodecJsonSonic encoding/json
func NewCodecJsonSonic[V any]() Codec[V] {
	return &jsonStd[V]{}
}

type jsonSonic[V any] struct{}

func (j *jsonSonic[V]) Marshal(v *V) ([]byte, error) {
	return sonic.Marshal(v)
}

func (j *jsonSonic[V]) Unmarshal(data []byte) (*V, error) {
	var v V
	err := sonic.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// NewCodecJsonStd encoding/json
func NewCodecJsonStd[V any]() Codec[V] {
	return &jsonStd[V]{}
}

type jsonStd[V any] struct{}

func (j *jsonStd[V]) Marshal(v *V) ([]byte, error) {
	return json.Marshal(v)
}

func (j *jsonStd[V]) Unmarshal(data []byte) (*V, error) {
	var v V
	err := json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
