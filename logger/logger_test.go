package logger

import (
	"os"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// / TestInit 测试初始化函数
func TestInit(t *testing.T) {
	resetGlobalState()

	t.Run("默认初始化", func(t *testing.T) {
		resetGlobalState()
		Init()
		assert.NotNil(t, globalLogger)
		assert.Equal(t, logrus.DebugLevel, globalLogger.GetLevel())
	})

	t.Run("包初始化成功", func(t *testing.T) {
		// 验证包导入时init函数是否执行成功
		assert.NotNil(t, globalLogger)
		assert.Equal(t, logrus.DebugLevel, globalLogger.GetLevel())
	})
}

// TestConfig 测试配置相关功能
func TestConfig(t *testing.T) {
	t.Run("默认配置", func(t *testing.T) {
		cfg := defaultConfig()
		assert.Equal(t, "", cfg.fileName)
		assert.Equal(t, logrus.DebugLevel, cfg.level)
		assert.Equal(t, 32, cfg.maxSize)
		assert.Equal(t, 5, cfg.maxBackups)
		assert.Equal(t, 30, cfg.maxAge)
		assert.False(t, cfg.compress)
		assert.False(t, cfg.jsonFormat)
		assert.True(t, cfg.withConsole)
	})

	t.Run("配置验证", func(t *testing.T) {
		testCases := []struct {
			name      string
			config    *config
			expectErr bool
		}{
			{"有效配置", &config{maxSize: 10, maxBackups: 5, maxAge: 30}, false},
			{"maxSize负数", &config{maxSize: -1, maxBackups: 5, maxAge: 30}, true},
			{"maxBackups负数", &config{maxSize: 10, maxBackups: -2, maxAge: 30}, true},
			{"maxBackups-1有效", &config{maxSize: 10, maxBackups: -1, maxAge: 30}, false},
			{"maxAge负数", &config{maxSize: 10, maxBackups: 5, maxAge: -1}, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validateConfig(tc.config)
				if tc.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

// TestNewLogger 测试创建logger
func TestNewLogger(t *testing.T) {
	t.Run("默认配置创建logger", func(t *testing.T) {
		resetGlobalState()
		logger, err := newLogger()
		require.NoError(t, err)
		assert.NotNil(t, logger)
		assert.Equal(t, logrus.DebugLevel, logger.GetLevel())
	})

	t.Run("文本格式logger", func(t *testing.T) {
		resetGlobalState()
		logger, err := newLogger(WithJSONFormat(false))
		require.NoError(t, err)
		_, ok := logger.Formatter.(*logrus.TextFormatter)
		assert.True(t, ok)
	})

	t.Run("JSON格式logger", func(t *testing.T) {
		resetGlobalState()
		logger, err := newLogger(WithJSONFormat(true))
		require.NoError(t, err)
		_, ok := logger.Formatter.(*logrus.JSONFormatter)
		assert.True(t, ok)
	})
}

// TestFallbackLogger 测试降级logger
func TestFallbackLogger(t *testing.T) {
	t.Run("创建降级logger", func(t *testing.T) {
		logger := createFallbackLogger()
		assert.NotNil(t, logger)
		assert.Equal(t, logrus.DebugLevel, logger.GetLevel())

		// 验证输出是标准输出
		assert.Equal(t, os.Stdout, logger.Out)
	})

	t.Run("验证降级logger格式", func(t *testing.T) {
		logger := createFallbackLogger()
		formatter, ok := logger.Formatter.(*logrus.TextFormatter)
		assert.True(t, ok)
		assert.True(t, formatter.FullTimestamp)
		assert.Equal(t, "2006-01-02 15:04:05", formatter.TimestampFormat)
		assert.True(t, formatter.ForceColors)
	})
}

// resetGlobalState 重置全局状态用于测试
func resetGlobalState() {
	once = sync.Once{}
	// 注意：不要将globalLogger设为nil，因为其他测试可能依赖它
}

// TestOptions 测试配置选项
func TestOptions(t *testing.T) {
	t.Run("WithFileName", func(t *testing.T) {
		cfg := defaultConfig()
		WithFileName("test.log")(cfg)
		assert.Equal(t, "test.log", cfg.fileName)
	})

	t.Run("WithLevel", func(t *testing.T) {
		cfg := defaultConfig()
		WithLevel(logrus.InfoLevel)(cfg)
		assert.Equal(t, logrus.InfoLevel, cfg.level)
	})

	t.Run("WithLevelString", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected logrus.Level
		}{
			{"debug", logrus.DebugLevel},
			{"info", logrus.InfoLevel},
			{"warn", logrus.WarnLevel},
			{"error", logrus.ErrorLevel},
			{"trace", logrus.TraceLevel},
			{"panic", logrus.PanicLevel},
			{"fatal", logrus.FatalLevel},
			{"invalid", logrus.DebugLevel}, // 无效输入保持默认
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				cfg := defaultConfig()
				WithLevelString(tc.input)(cfg)
				assert.Equal(t, tc.expected, cfg.level)
			})
		}
	})

	t.Run("WithJSONFormat", func(t *testing.T) {
		cfg := defaultConfig()
		WithJSONFormat(true)(cfg)
		assert.True(t, cfg.jsonFormat)
	})

	t.Run("WithConsoleOutput", func(t *testing.T) {
		cfg := defaultConfig()
		WithConsoleOutput(false)(cfg)
		assert.False(t, cfg.withConsole)
	})

	t.Run("WithMaxSize", func(t *testing.T) {
		cfg := defaultConfig()
		WithMaxSize(100)(cfg)
		assert.Equal(t, 100, cfg.maxSize)

		// 测试负数不改变配置
		cfg = defaultConfig()
		WithMaxSize(-1)(cfg)
		assert.Equal(t, 32, cfg.maxSize) // 保持默认值
	})

	t.Run("WithMaxBackups", func(t *testing.T) {
		cfg := defaultConfig()
		WithMaxBackups(10)(cfg)
		assert.Equal(t, 10, cfg.maxBackups)

		// 测试-1有效
		cfg = defaultConfig()
		WithMaxBackups(-1)(cfg)
		assert.Equal(t, -1, cfg.maxBackups)
	})

	t.Run("WithMaxAge", func(t *testing.T) {
		cfg := defaultConfig()
		WithMaxAge(7)(cfg)
		assert.Equal(t, 7, cfg.maxAge)

		// 测试负数不改变配置
		cfg = defaultConfig()
		WithMaxAge(-1)(cfg)
		assert.Equal(t, 30, cfg.maxAge) // 保持默认值
	})

	t.Run("WithCompress", func(t *testing.T) {
		cfg := defaultConfig()
		WithCompress(true)(cfg)
		assert.True(t, cfg.compress)
	})
}

// TestEdgeCases 测试边界情况
func TestEdgeCases(t *testing.T) {
	t.Run("空文件名配置", func(t *testing.T) {
		resetGlobalState()
		logger, err := newLogger(
			WithFileName(""),
			WithConsoleOutput(false),
		)
		require.NoError(t, err)
		require.NotNil(t, logger)

		// 当文件名为空时，应该输出到标准输出
		assert.Equal(t, os.Stdout, logger.Out)
	})

	t.Run("无效日志级别字符串", func(t *testing.T) {
		cfg := defaultConfig()
		WithLevelString("invalid-level")(cfg)
		// 应该保持默认的DebugLevel
		assert.Equal(t, logrus.DebugLevel, cfg.level)
	})

	t.Run("大小写不敏感的日志级别", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected logrus.Level
		}{
			{"DEBUG", logrus.DebugLevel},
			{"Info", logrus.InfoLevel},
			{"WARN", logrus.WarnLevel},
			{"ErRoR", logrus.ErrorLevel},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				cfg := defaultConfig()
				WithLevelString(tc.input)(cfg)
				assert.Equal(t, tc.expected, cfg.level)
			})
		}
	})

	t.Run("日志级别转换", func(t *testing.T) {
		cfg := defaultConfig()
		assert.Equal(t, logrus.DebugLevel, cfg.level)

		WithLevel(logrus.ErrorLevel)(cfg)
		assert.Equal(t, logrus.ErrorLevel, cfg.level)

		WithLevelString("warn")(cfg)
		assert.Equal(t, logrus.WarnLevel, cfg.level)
	})
}
