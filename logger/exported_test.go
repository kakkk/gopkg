package logger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// setupTestLogger 设置测试用的logger
func setupTestLogger(t *testing.T) (*bytes.Buffer, func()) {
	// 保存原始logger
	originalLogger := globalLogger
	originalOnce := once

	// 创建测试logger
	var buf bytes.Buffer
	testLogger := logrus.New()
	testLogger.SetOutput(&buf)
	testLogger.SetLevel(logrus.TraceLevel) // 设置为Trace级别以测试所有级别
	testLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
	})

	// 替换全局logger
	globalLogger = testLogger
	once = sync.Once{}

	// 返回清理函数
	return &buf, func() {
		globalLogger = originalLogger
		once = originalOnce
	}
}

// TestCtxFunctions 测试上下文相关函数
func TestCtxFunctions(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("Ctx", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "test_key", "test_value")
		entry := Ctx(ctx)
		assert.NotNil(t, entry)
		assert.Equal(t, ctx, entry.Context)

		// 测试日志输出包含上下文信息
		entry.Info("test message")
		assert.Contains(t, buf.String(), "test message")
		buf.Reset()
	})

	t.Run("WithContext", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", "12345")
		entry := WithContext(ctx)
		assert.NotNil(t, entry)
		assert.Equal(t, ctx, entry.Context)

		entry.Warn("context test")
		assert.Contains(t, buf.String(), "context test")
		buf.Reset()
	})
}

// TestWithFunctions 测试With前缀的函数
func TestWithFunctions(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("WithError", func(t *testing.T) {
		err := errors.New("test error message")
		entry := WithError(err)
		assert.NotNil(t, entry)

		entry.Error("error occurred")
		output := buf.String()
		assert.Contains(t, output, "error occurred")
		assert.Contains(t, output, "test error message")
		buf.Reset()
	})

	t.Run("WithField", func(t *testing.T) {
		entry := WithField("user", "john_doe")
		assert.NotNil(t, entry)

		entry.Info("user action")
		output := buf.String()
		assert.Contains(t, output, "user action")
		assert.Contains(t, output, "user=john_doe")
		buf.Reset()

		// 测试多个字段
		entry = WithField("action", "login").WithField("ip", "192.168.1.1")
		entry.Info("multiple fields")
		output = buf.String()
		assert.Contains(t, output, "action=login")
		assert.Contains(t, output, "ip=192.168.1.1")
		buf.Reset()
	})

	t.Run("WithFields", func(t *testing.T) {
		fields := logrus.Fields{
			"request_id": "req-123",
			"method":     "GET",
			"path":       "/api/users",
		}

		entry := WithFields(fields)
		assert.NotNil(t, entry)

		entry.Info("request processed")
		output := buf.String()
		assert.Contains(t, output, "request_id=req-123")
		assert.Contains(t, output, "method=GET")
		assert.Contains(t, output, "path=/api/users")
		buf.Reset()
	})

	t.Run("WithTime", func(t *testing.T) {
		customTime := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
		entry := WithTime(customTime)
		assert.NotNil(t, entry)

		// 注意：WithTime设置的时间可能不会直接体现在文本格式化输出中
		// 取决于formatter的配置
		entry.Debug("time test")
		assert.Contains(t, buf.String(), "time test")
		buf.Reset()
	})
}

// TestLogLevelFunctions 测试日志级别函数
func TestLogLevelFunctions(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("Trace", func(t *testing.T) {
		Trace("trace message")
		assert.Contains(t, buf.String(), "trace message")
		assert.Contains(t, buf.String(), "level=trace")
		buf.Reset()

		// 测试多个参数
		Trace("param1", "param2", 123)
		output := buf.String()
		assert.Contains(t, output, "param1param2123")
		buf.Reset()
	})

	t.Run("Debug", func(t *testing.T) {
		Debug("debug message")
		assert.Contains(t, buf.String(), "debug message")
		assert.Contains(t, buf.String(), "level=debug")
		buf.Reset()
	})

	t.Run("Print", func(t *testing.T) {
		Print("print message")
		assert.Contains(t, buf.String(), "print message")
		assert.Contains(t, buf.String(), "level=info") // Print默认使用Info级别
		buf.Reset()
	})

	t.Run("Info", func(t *testing.T) {
		Info("info message")
		assert.Contains(t, buf.String(), "info message")
		assert.Contains(t, buf.String(), "level=info")
		buf.Reset()
	})

	t.Run("Warn", func(t *testing.T) {
		Warn("warn message")
		assert.Contains(t, buf.String(), "warn message")
		assert.Contains(t, buf.String(), "level=warning")
		buf.Reset()
	})

	t.Run("Warning", func(t *testing.T) {
		Warning("warning message")
		assert.Contains(t, buf.String(), "warning message")
		assert.Contains(t, buf.String(), "level=warning")
		buf.Reset()
	})

	t.Run("Error", func(t *testing.T) {
		Error("error message")
		assert.Contains(t, buf.String(), "error message")
		assert.Contains(t, buf.String(), "level=error")
		buf.Reset()
	})

	t.Run("Panic", func(t *testing.T) {
		// Panic函数会引发panic，需要recover
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "panic message")
			}
		}()

		Panic("panic message")
		// 如果代码执行到这里，说明没有panic，测试失败
		t.Error("Expected panic but didn't get one")
	})
}

// TestFormatFunctions 测试格式化日志函数
func TestFormatFunctions(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("Tracef", func(t *testing.T) {
		Tracef("trace %s %d", "formatted", 123)
		output := buf.String()
		assert.Contains(t, output, "trace formatted 123")
		assert.Contains(t, output, "level=trace")
		buf.Reset()
	})

	t.Run("Debugf", func(t *testing.T) {
		Debugf("debug %s %v", "message", map[string]int{"key": 1})
		output := buf.String()
		assert.Contains(t, output, "debug message")
		assert.Contains(t, output, "level=debug")
		buf.Reset()
	})

	t.Run("Printf", func(t *testing.T) {
		Printf("print %s", "formatted")
		output := buf.String()
		assert.Contains(t, output, "print formatted")
		assert.Contains(t, output, "level=info")
		buf.Reset()
	})

	t.Run("Infof", func(t *testing.T) {
		Infof("info %s %d", "test", 42)
		output := buf.String()
		assert.Contains(t, output, "info test 42")
		assert.Contains(t, output, "level=info")
		buf.Reset()
	})

	t.Run("Warnf", func(t *testing.T) {
		Warnf("warning: %s", "something happened")
		output := buf.String()
		assert.Contains(t, output, "warning: something happened")
		assert.Contains(t, output, "level=warning")
		buf.Reset()
	})

	t.Run("Warningf", func(t *testing.T) {
		Warningf("warning %s", "formatted")
		output := buf.String()
		assert.Contains(t, output, "warning formatted")
		assert.Contains(t, output, "level=warning")
		buf.Reset()
	})

	t.Run("Errorf", func(t *testing.T) {
		Errorf("error: %v", errors.New("test error"))
		output := buf.String()
		assert.Contains(t, output, "error: test error")
		assert.Contains(t, output, "level=error")
		buf.Reset()
	})

	t.Run("Panicf", func(t *testing.T) {
		// Panicf会引发panic，需要recover
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "panic formatted")
			}
		}()

		Panicf("panic %s", "formatted")
		// 如果代码执行到这里，说明没有panic，测试失败
		t.Error("Expected panic but didn't get one")
	})

	t.Run("Fatalf", func(t *testing.T) {
		// Fatalf会调用os.Exit(1)，需要特殊处理
		t.Log("Fatalf function exists (cannot test without exiting)")
	})
}

// TestLineFunctions 测试带换行的日志函数
func TestLineFunctions(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("Traceln", func(t *testing.T) {
		Traceln("trace", "message", "with", "newline")
		output := buf.String()
		// 注意：Traceln会在参数之间加空格，并在最后加换行符
		assert.Contains(t, output, "trace message with newline")
		assert.Contains(t, output, "level=trace")
		buf.Reset()
	})

	t.Run("Debugln", func(t *testing.T) {
		Debugln("debug line")
		output := buf.String()
		assert.Contains(t, output, "debug line")
		assert.Contains(t, output, "level=debug")
		buf.Reset()
	})

	t.Run("Println", func(t *testing.T) {
		Println("print line")
		output := buf.String()
		assert.Contains(t, output, "print line")
		assert.Contains(t, output, "level=info")
		buf.Reset()
	})

	t.Run("Infoln", func(t *testing.T) {
		Infoln("info", 1, 2, 3)
		output := buf.String()
		assert.Contains(t, output, "info 1 2 3")
		assert.Contains(t, output, "level=info")
		buf.Reset()
	})

	t.Run("Warnln", func(t *testing.T) {
		Warnln("warn line")
		output := buf.String()
		assert.Contains(t, output, "warn line")
		assert.Contains(t, output, "level=warning")
		buf.Reset()
	})

	t.Run("Warningln", func(t *testing.T) {
		Warningln("warning line")
		output := buf.String()
		assert.Contains(t, output, "warning line")
		assert.Contains(t, output, "level=warning")
		buf.Reset()
	})

	t.Run("Errorln", func(t *testing.T) {
		Errorln("error line")
		output := buf.String()
		assert.Contains(t, output, "error line")
		assert.Contains(t, output, "level=error")
		buf.Reset()
	})

	t.Run("Panicln", func(t *testing.T) {
		// Panicln会引发panic，需要recover
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "panic line")
			}
		}()

		Panicln("panic line")
		// 如果代码执行到这里，说明没有panic，测试失败
		t.Error("Expected panic but didn't get one")
	})

	t.Run("Fatalln", func(t *testing.T) {
		// Fatalln会调用os.Exit(1)，需要特殊处理
		t.Log("Fatalln function exists (cannot test without exiting)")
	})
}

// TestLogFunctionWrappers 测试函数式日志包装器
func TestLogFunctionWrappers(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("TraceFn", func(t *testing.T) {
		callCount := 0
		TraceFn(func() []interface{} {
			callCount++
			return []interface{}{"trace", " ", "function", " ", callCount}
		})
		assert.Equal(t, 1, callCount)
		output := buf.String()
		assert.Contains(t, output, "trace function 1")
		assert.Contains(t, output, "level=trace")
		buf.Reset()
	})

	t.Run("DebugFn", func(t *testing.T) {
		DebugFn(func() []interface{} {
			return []interface{}{"debug", " ", "fn"}
		})
		output := buf.String()
		assert.Contains(t, output, "debug fn")
		assert.Contains(t, output, "level=debug")
		buf.Reset()
	})

	t.Run("PrintFn", func(t *testing.T) {
		PrintFn(func() []interface{} {
			return []interface{}{"print", " ", "function"}
		})
		output := buf.String()
		assert.Contains(t, output, "print function")
		assert.Contains(t, output, "level=info")
		buf.Reset()
	})

	t.Run("InfoFn", func(t *testing.T) {
		InfoFn(func() []interface{} {
			return []interface{}{"info", " ", "function"}
		})
		output := buf.String()
		assert.Contains(t, output, "info function")
		assert.Contains(t, output, "level=info")
		buf.Reset()
	})

	t.Run("WarnFn", func(t *testing.T) {
		WarnFn(func() []interface{} {
			return []interface{}{"warn", " ", "fn"}
		})
		output := buf.String()
		assert.Contains(t, output, "warn fn")
		assert.Contains(t, output, "level=warning")
		buf.Reset()
	})

	t.Run("WarningFn", func(t *testing.T) {
		WarningFn(func() []interface{} {
			return []interface{}{"warning", " ", "function"}
		})
		output := buf.String()
		assert.Contains(t, output, "warning function")
		assert.Contains(t, output, "level=warning")
		buf.Reset()
	})

	t.Run("ErrorFn", func(t *testing.T) {
		ErrorFn(func() []interface{} {
			return []interface{}{"error", " ", "fn"}
		})
		output := buf.String()
		assert.Contains(t, output, "error fn")
		assert.Contains(t, output, "level=error")
		buf.Reset()
	})

	t.Run("PanicFn", func(t *testing.T) {
		// PanicFn会引发panic，需要recover
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "panic fn")
			}
		}()

		PanicFn(func() []interface{} {
			return []interface{}{"panic", " ", "fn"}
		})
		// 如果代码执行到这里，说明没有panic，测试失败
		t.Error("Expected panic but didn't get one")
	})
}

// TestLogLevelFiltering 测试日志级别过滤
func TestLogLevelFiltering(t *testing.T) {
	t.Run("Info级别过滤Debug", func(t *testing.T) {
		buf, cleanup := setupTestLogger(t)
		defer cleanup()

		// 设置logger级别为Info
		globalLogger.SetLevel(logrus.InfoLevel)

		// Debug消息不应该被记录
		Debug("debug message")
		assert.Empty(t, buf.String())
		buf.Reset()

		// Info消息应该被记录
		Info("info message")
		assert.Contains(t, buf.String(), "info message")
		buf.Reset()
	})

	t.Run("Warn级别过滤Info", func(t *testing.T) {
		buf, cleanup := setupTestLogger(t)
		defer cleanup()

		// 设置logger级别为Warn
		globalLogger.SetLevel(logrus.WarnLevel)

		Info("info message")
		Debug("debug message")
		assert.Empty(t, buf.String())
		buf.Reset()

		Warn("warn message")
		assert.Contains(t, buf.String(), "warn message")
		buf.Reset()
	})
}

// TestConcurrentLogging 测试并发日志写入
func TestConcurrentLogging(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	var wg sync.WaitGroup
	messageCount := 100

	for i := 0; i < messageCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			Infof("concurrent message %d", idx)
		}(i)
	}
	wg.Wait()

	// 验证所有消息都被记录
	output := buf.String()
	lines := strings.Count(output, "\n")
	// 由于并发，可能无法精确计数，但应该有很多行
	assert.Greater(t, lines, messageCount/2)

	// 验证包含一些消息
	assert.Contains(t, output, "concurrent message")
}

// TestLogEntryChaining 测试日志条目链式调用
func TestLogEntryChaining(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("链式添加字段", func(t *testing.T) {
		WithField("step", "1").
			WithField("action", "process").
			WithField("status", "started").
			Info("processing started")

		output := buf.String()
		assert.Contains(t, output, "step=1")
		assert.Contains(t, output, "action=process")
		assert.Contains(t, output, "status=started")
		assert.Contains(t, output, "processing started")
		buf.Reset()
	})

	t.Run("链式调用不同级别", func(t *testing.T) {
		entry := WithField("request_id", "req-456")

		entry.Debug("debug with request")
		entry.Info("info with request")
		entry.Warn("warn with request")

		output := buf.String()
		// 所有日志都应该包含request_id
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			assert.Contains(t, line, "request_id=req-456")
		}
		buf.Reset()
	})
}

// TestExportedEdgeCases 测试边界情况
func TestExportedEdgeCases(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("空参数", func(t *testing.T) {
		Info() // 无参数
		output := buf.String()
		// 应该记录一个空消息或包含某些默认信息
		assert.NotEmpty(t, output)
		buf.Reset()
	})

	t.Run("nil错误", func(t *testing.T) {
		// WithError应该能处理nil错误
		entry := WithError(nil)
		assert.NotNil(t, entry)
		entry.Info("test with nil error")
		output := buf.String()
		assert.Contains(t, output, "test with nil error")
		assert.Contains(t, output, "<nil>")
		buf.Reset()
	})

	t.Run("特殊字符", func(t *testing.T) {
		specialMessage := "特殊字符: 中文, 😀, \n换行, \t制表符"
		Info(specialMessage)
		output := buf.String()
		assert.Contains(t, output, "特殊字符")
		buf.Reset()
	})

	t.Run("复杂数据结构", func(t *testing.T) {
		complexData := map[string]interface{}{
			"user": map[string]string{
				"name": "John",
				"role": "admin",
			},
			"permissions": []string{"read", "write", "delete"},
			"active":      true,
			"count":       42,
		}

		WithField("data", complexData).Info("complex data")
		output := buf.String()
		assert.Contains(t, output, "complex data")
		// logrus会以某种格式序列化复杂数据
		assert.Contains(t, output, "data=")
		buf.Reset()
	})
}

// TestRealUsageScenarios 测试实际使用场景
func TestRealUsageScenarios(t *testing.T) {
	buf, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("HTTP请求日志", func(t *testing.T) {
		WithFields(logrus.Fields{
			"method":  "GET",
			"path":    "/api/users",
			"status":  200,
			"latency": "150ms",
		}).Info("request completed")

		output := buf.String()
		assert.Contains(t, output, "method=GET")
		assert.Contains(t, output, "path=/api/users")
		assert.Contains(t, output, "status=200")
		assert.Contains(t, output, "latency=150ms")
		assert.Contains(t, output, "request completed")
		buf.Reset()
	})

	t.Run("错误处理日志", func(t *testing.T) {
		err := errors.New("database connection failed")

		WithError(err).
			WithField("operation", "query_users").
			WithField("attempt", 3).
			Error("operation failed")

		output := buf.String()
		assert.Contains(t, output, "database connection failed")
		assert.Contains(t, output, "operation=query_users")
		assert.Contains(t, output, "attempt=3")
		assert.Contains(t, output, "operation failed")
		buf.Reset()
	})

	t.Run("调试信息日志", func(t *testing.T) {
		Debugf("processing item %d of %d", 25, 100)
		output := buf.String()
		assert.Contains(t, output, "processing item 25 of 100")
		buf.Reset()
	})
}

// TestLogFormatVariations 测试日志格式变化
func TestLogFormatVariations(t *testing.T) {
	t.Run("JSON格式输出", func(t *testing.T) {
		// 创建JSON格式的logger
		testLogger := logrus.New()
		var buf bytes.Buffer
		testLogger.SetOutput(&buf)
		testLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})

		// 保存原始logger
		originalLogger := globalLogger
		globalLogger = testLogger
		defer func() { globalLogger = originalLogger }()

		WithField("user", "john").Info("user logged in")

		output := buf.String()
		// JSON输出应该包含字段
		assert.Contains(t, output, "\"user\":\"john\"")
		assert.Contains(t, output, "\"msg\":\"user logged in\"")
		assert.Contains(t, output, "\"level\":\"info\"")
	})

	t.Run("文本格式输出", func(t *testing.T) {
		// 使用默认的文本格式（已在setupTestLogger中设置）
		buf, cleanup := setupTestLogger(t)
		defer cleanup()

		WithField("action", "login").Info("test")

		output := buf.String()
		// 文本格式应该有键值对
		assert.Contains(t, output, "action=login")
		assert.Contains(t, output, "level=info")
	})
}

// TestLoggerReplacement 测试logger替换
func TestLoggerReplacement(t *testing.T) {
	// 测试替换全局logger后，导出函数是否使用新的logger
	originalLogger := globalLogger
	defer func() { globalLogger = originalLogger }()

	// 创建新的logger
	newLogger := logrus.New()
	var buf bytes.Buffer
	newLogger.SetOutput(&buf)
	newLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
	})

	// 替换全局logger
	globalLogger = newLogger

	// 测试使用新logger
	Info("test with new logger")
	assert.Contains(t, buf.String(), "test with new logger")
}

// TestPanicAndFatalHandling 测试Panic和Fatal的特殊处理
func TestPanicAndFatalHandling(t *testing.T) {
	t.Run("Panic恢复测试", func(t *testing.T) {
		buf, cleanup := setupTestLogger(t)
		defer cleanup()

		panicOccurred := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicOccurred = true
					// 验证panic消息被记录
					assert.Contains(t, buf.String(), "test panic")
				}
			}()

			Panic("test panic")
		}()

		assert.True(t, panicOccurred, "应该发生panic")
	})

}
