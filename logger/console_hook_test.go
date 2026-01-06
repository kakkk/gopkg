package logger

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestConsoleHook 测试控制台hook
func TestConsoleHook(t *testing.T) {
	t.Run("控制台hook格式化", func(t *testing.T) {
		formatter := getConsoleFormatter(false)
		_, ok := formatter.(*logrus.TextFormatter)
		assert.True(t, ok)

		formatter = getConsoleFormatter(true)
		_, ok = formatter.(*logrus.JSONFormatter)
		assert.True(t, ok)
	})

	t.Run("consoleHook实现", func(t *testing.T) {
		hook := &consoleHook{
			formatter: &logrus.TextFormatter{
				DisableColors:    true,
				DisableTimestamp: true,
			},
		}

		// 测试Levels方法
		levels := hook.Levels()
		assert.Equal(t, logrus.AllLevels, levels)

		// 测试Fire方法
		entry := &logrus.Entry{
			Logger:  logrus.New(),
			Time:    time.Now(),
			Level:   logrus.InfoLevel,
			Message: "test message",
			Data:    logrus.Fields{"key": "value"},
		}

		// 捕获标准输出
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := hook.Fire(entry)

		// 关闭写入端，读取输出
		w.Close()
		output, _ := io.ReadAll(r)
		os.Stdout = oldStdout

		assert.NoError(t, err)
		assert.Contains(t, string(output), "test message")
		assert.Contains(t, string(output), "key=value")
	})

	t.Run("consoleHook多级别", func(t *testing.T) {
		hook := &consoleHook{
			formatter: &logrus.TextFormatter{
				DisableColors:    true,
				DisableTimestamp: true,
			},
		}

		levels := hook.Levels()
		assert.Contains(t, levels, logrus.DebugLevel)
		assert.Contains(t, levels, logrus.InfoLevel)
		assert.Contains(t, levels, logrus.WarnLevel)
		assert.Contains(t, levels, logrus.ErrorLevel)
		assert.Contains(t, levels, logrus.FatalLevel)
		assert.Contains(t, levels, logrus.PanicLevel)
	})
}

// TestAddConsoleHook 测试添加控制台hook
func TestAddConsoleHook(t *testing.T) {
	t.Run("添加文本格式控制台hook", func(t *testing.T) {
		logger := logrus.New()
		addConsoleHook(logger, false)

		hasConsoleHook := false
		for _, hooks := range logger.Hooks {
			for _, hook := range hooks {
				if _, ok := hook.(*consoleHook); ok {
					hasConsoleHook = true
					break
				}
			}
		}
		assert.True(t, hasConsoleHook)
	})

	t.Run("添加JSON格式控制台hook", func(t *testing.T) {
		logger := logrus.New()
		addConsoleHook(logger, true)

		hasConsoleHook := false
		for _, hooks := range logger.Hooks {
			for _, hook := range hooks {
				if _, ok := hook.(*consoleHook); ok {
					hasConsoleHook = true
					break
				}
			}
		}
		assert.True(t, hasConsoleHook)
	})
}
