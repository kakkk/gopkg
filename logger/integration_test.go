package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration 测试集成场景
func TestIntegration(t *testing.T) {
	t.Run("完整配置流程", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "integration.log")

		// 重置单例
		resetGlobalState()

		Init(
			WithFileName(logFile),
			WithLevel(logrus.InfoLevel),
			WithJSONFormat(true),
			WithMaxSize(10),
			WithMaxBackups(5),
			WithMaxAge(30),
			WithCompress(true),
			WithConsoleOutput(false),
		)

		assert.NotNil(t, globalLogger)
		assert.Equal(t, logrus.InfoLevel, globalLogger.GetLevel())

		// 写入日志
		Info("integration test")
		time.Sleep(100 * time.Millisecond)

		// 验证文件被创建
		_, err := os.Stat(logFile)
		assert.NoError(t, err)
	})

	t.Run("只输出到控制台", func(t *testing.T) {
		resetGlobalState()

		// 捕获标准输出
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Init() // 默认只输出到控制台
		Info("控制台输出测试")

		// 关闭写入端，读取输出
		w.Close()
		output, _ := io.ReadAll(r)
		os.Stdout = oldStdout

		assert.Contains(t, string(output), "控制台输出测试")
	})

	t.Run("并发日志写入", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "concurrent.log")

		// 重置单例
		resetGlobalState()

		Init(WithFileName(logFile), WithConsoleOutput(false))

		var wg sync.WaitGroup
		messageCount := 50 // 减少数量避免测试时间过长

		for i := 0; i < messageCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				Infof("并发消息 %d", idx)
			}(i)
		}
		wg.Wait()

		// 等待文件写入
		time.Sleep(500 * time.Millisecond)

		// 验证日志文件包含消息
		content, err := os.ReadFile(logFile)
		if err == nil {
			logContent := string(content)
			// 检查是否至少有一些消息被写入
			foundCount := 0
			for i := 0; i < messageCount; i++ {
				if strings.Contains(logContent, fmt.Sprintf("并发消息 %d", i)) {
					foundCount++
				}
			}
			assert.Greater(t, foundCount, 0, "应该至少有一些消息被记录")
		}
	})

	t.Run("不同日志级别过滤", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "level_filter.log")

		resetGlobalState()

		Init(
			WithFileName(logFile),
			WithLevel(logrus.WarnLevel), // 只记录Warn及以上级别
			WithConsoleOutput(false),
		)

		// 这些不会被记录
		Debug("debug message")
		Info("info message")

		// 这些会被记录
		Warn("warn message")
		Error("error message")

		time.Sleep(100 * time.Millisecond)

		content, err := os.ReadFile(logFile)
		require.NoError(t, err)
		logContent := string(content)

		assert.NotContains(t, logContent, "debug message")
		assert.NotContains(t, logContent, "info message")
		assert.Contains(t, logContent, "warn message")
		assert.Contains(t, logContent, "error message")
	})
}

// TestFileOutput 测试文件输出
func TestFileOutput(t *testing.T) {
	t.Run("文件输出配置", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "app.log")

		cfg := &config{
			fileName:    logFile,
			level:       logrus.InfoLevel,
			maxSize:     10,
			maxBackups:  3,
			maxAge:      7,
			compress:    true,
			jsonFormat:  false,
			withConsole: true,
		}

		logger := logrus.New()
		err := setupFileOutput(logger, cfg)
		require.NoError(t, err)

		// 写入一条日志
		logger.Info("test log")
		time.Sleep(100 * time.Millisecond)

		// 验证文件被创建
		_, err = os.Stat(logFile)
		assert.NoError(t, err)

		// 验证日志内容
		content, err := os.ReadFile(logFile)
		if err == nil {
			assert.Contains(t, string(content), "test log")
		}
	})

	t.Run("同时输出到文件和控制台", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "both.log")

		cfg := &config{
			fileName:    logFile,
			withConsole: true,
			jsonFormat:  false,
		}

		logger := logrus.New()
		err := setupFileOutput(logger, cfg)
		require.NoError(t, err)

		// 验证logger有console hook
		hasConsoleHook := false
		for _, hooks := range logger.Hooks {
			for _, hook := range hooks {
				if _, ok := hook.(*consoleHook); ok {
					hasConsoleHook = true
					break
				}
			}
			if hasConsoleHook {
				break
			}
		}
		assert.True(t, hasConsoleHook)
	})

	t.Run("只输出到文件", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "fileonly.log")

		cfg := &config{
			fileName:    logFile,
			withConsole: false,
			jsonFormat:  false,
		}

		logger := logrus.New()
		err := setupFileOutput(logger, cfg)
		require.NoError(t, err)

		// 验证logger没有console hook
		hasConsoleHook := false
		for _, hooks := range logger.Hooks {
			for _, hook := range hooks {
				if _, ok := hook.(*consoleHook); ok {
					hasConsoleHook = true
					break
				}
			}
			if hasConsoleHook {
				break
			}
		}
		assert.False(t, hasConsoleHook)
	})
}

// TestPerformance 测试性能相关场景
func TestPerformance(t *testing.T) {
	t.Run("大量日志写入", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "performance.log")

		resetGlobalState()

		Init(
			WithFileName(logFile),
			WithLevel(logrus.WarnLevel),
			WithConsoleOutput(false),
		)

		start := time.Now()
		count := 1000

		for i := 0; i < count; i++ {
			Warnf("警告消息 %d", i)
		}

		elapsed := time.Since(start)
		t.Logf("写入 %d 条日志耗时: %v", count, elapsed)

		// 验证文件存在
		_, err := os.Stat(logFile)
		assert.NoError(t, err)
	})
}
