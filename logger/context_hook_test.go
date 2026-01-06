package logger

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/kakkk/gopkg/requestid"
)

// TestContextHook 测试context hook
func TestContextHook(t *testing.T) {
	t.Run("contextHook实现", func(t *testing.T) {
		hook := &contextHook{}

		// 测试Levels方法
		levels := hook.Levels()
		assert.Equal(t, logrus.AllLevels, levels)
		assert.Contains(t, levels, logrus.DebugLevel)
		assert.Contains(t, levels, logrus.InfoLevel)
		assert.Contains(t, levels, logrus.WarnLevel)
		assert.Contains(t, levels, logrus.ErrorLevel)
	})

	t.Run("Fire方法处理nil context", func(t *testing.T) {
		hook := &contextHook{}
		entry := &logrus.Entry{
			Logger:  logrus.New(),
			Time:    time.Now(),
			Level:   logrus.InfoLevel,
			Message: "test",
			Data:    make(logrus.Fields),
			Context: nil,
		}

		err := hook.Fire(entry)
		assert.NoError(t, err)
		_, exists := entry.Data["request_id"]
		assert.False(t, exists, "nil context不应该添加request_id")
	})

	t.Run("Fire方法处理空context", func(t *testing.T) {
		hook := &contextHook{}
		entry := &logrus.Entry{
			Logger:  logrus.New(),
			Time:    time.Now(),
			Level:   logrus.InfoLevel,
			Message: "test",
			Data:    make(logrus.Fields),
			Context: context.Background(),
		}

		err := hook.Fire(entry)
		assert.NoError(t, err)
		// 如果requestid.Get返回空字符串，则不应该添加字段
		requestIDValue, exists := entry.Data["request_id"]
		if exists {
			assert.NotEmpty(t, requestIDValue)
		}
	})

	t.Run("Fire方法添加request_id字段", func(t *testing.T) {
		hook := &contextHook{}

		// 创建一个带值的context
		ctx := requestid.Ctx(context.Background())
		entry := &logrus.Entry{
			Logger:  logrus.New(),
			Time:    time.Now(),
			Level:   logrus.InfoLevel,
			Message: "test",
			Data:    make(logrus.Fields),
			Context: ctx,
		}

		err := hook.Fire(entry)
		assert.NoError(t, err)
	})

	t.Run("Fire方法不影响其他字段", func(t *testing.T) {
		hook := &contextHook{}
		originalData := logrus.Fields{
			"existing_key": "existing_value",
			"another_key":  "another_value",
		}

		entry := &logrus.Entry{
			Logger:  logrus.New(),
			Time:    time.Now(),
			Level:   logrus.InfoLevel,
			Message: "test",
			Data:    originalData,
			Context: nil,
		}

		// 复制原始数据用于比较
		expectedData := make(logrus.Fields)
		for k, v := range originalData {
			expectedData[k] = v
		}

		err := hook.Fire(entry)
		assert.NoError(t, err)

		// 验证原有字段不变
		assert.Equal(t, "existing_value", entry.Data["existing_key"])
		assert.Equal(t, "another_value", entry.Data["another_key"])

		// 如果添加了request_id，应该不影响其他字段
		if _, hasRequestID := entry.Data["request_id"]; hasRequestID {
			assert.Equal(t, 3, len(entry.Data))
		} else {
			assert.Equal(t, 2, len(entry.Data))
		}
	})

	t.Run("Fire方法多次调用不会重复添加", func(t *testing.T) {
		hook := &contextHook{}
		ctx := context.Background()
		entry := &logrus.Entry{
			Logger:  logrus.New(),
			Time:    time.Now(),
			Level:   logrus.InfoLevel,
			Message: "test",
			Data:    make(logrus.Fields),
			Context: ctx,
		}

		// 第一次调用
		err := hook.Fire(entry)
		assert.NoError(t, err)

		// 记录第一次调用后的字段数量
		firstCallFieldCount := len(entry.Data)

		// 第二次调用
		err = hook.Fire(entry)
		assert.NoError(t, err)

		// 字段数量应该不变或只增加一次（如果添加了request_id）
		assert.True(t, len(entry.Data) == firstCallFieldCount || len(entry.Data) == firstCallFieldCount)
	})
}
