package gormlogger

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kakkk/gopkg/logger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"
)

const testLogFile = "gorm_test.log"

func TestMain(m *testing.M) {
	// Initialize logger to write to a file for testing
	logger.Init(
		logger.WithFileName(testLogFile),
		logger.WithConsoleOutput(false),
		logger.WithJSONFormat(false),
		logger.WithLevel(logrus.DebugLevel), // Ensure we capture everything
	)

	code := m.Run()

	// Cleanup
	os.Remove(testLogFile)
	os.Exit(code)
}

func readAndClearLog() string {
	content, err := ioutil.ReadFile(testLogFile)
	if err != nil {
		return ""
	}
	// Clear the file
	os.Truncate(testLogFile, 0)
	return string(content)
}

func TestNew(t *testing.T) {
	l := New(WithSlowThreshold(100 * time.Millisecond))
	assert.NotNil(t, l)
}

func TestLogMode(t *testing.T) {
	l := New()
	l2 := l.LogMode(gLogger.Info)
	assert.NotSame(t, l, l2)
}

func TestTrace(t *testing.T) {
	// Clean log before starting
	readAndClearLog()

	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name          string
		logLevel      gLogger.LogLevel
		slowThreshold time.Duration
		ignoreRNF     bool
		err           error
		elapsed       time.Duration
		rows          int64
		sql           string
		expectLog     bool
		expectContent string
		expectLevel   string // "info", "warn", "error"
	}{
		{
			name:          "Info Level - Normal SQL",
			logLevel:      gLogger.Info,
			elapsed:       10 * time.Millisecond,
			sql:           "SELECT * FROM users",
			expectLog:     true,
			expectLevel:   "info",
			expectContent: "SELECT * FROM users",
		},
		{
			name:      "Warn Level - Normal SQL",
			logLevel:  gLogger.Warn,
			elapsed:   10 * time.Millisecond,
			sql:       "SELECT * FROM users",
			expectLog: false,
		},
		{
			name:          "Warn Level - Slow SQL",
			logLevel:      gLogger.Warn,
			slowThreshold: 100 * time.Millisecond,
			elapsed:       200 * time.Millisecond,
			sql:           "SELECT * FROM huge_table",
			expectLog:     true,
			expectLevel:   "warn",
			expectContent: "SELECT * FROM huge_table",
		},
		{
			name:          "Error Level - Error SQL",
			logLevel:      gLogger.Error,
			err:           errors.New("db error"),
			elapsed:       10 * time.Millisecond,
			sql:           "SELECT * FROM users",
			expectLog:     true,
			expectLevel:   "error",
			expectContent: "db error",
		},
		{
			name:      "Record Not Found - Ignore True",
			logLevel:  gLogger.Error,
			ignoreRNF: true,
			err:       gorm.ErrRecordNotFound,
			sql:       "SELECT * FROM users WHERE id=999",
			expectLog: false,
		},
		{
			name:          "Record Not Found - Ignore False",
			logLevel:      gLogger.Error,
			ignoreRNF:     false,
			err:           gorm.ErrRecordNotFound,
			sql:           "SELECT * FROM users WHERE id=999",
			expectLog:     true,
			expectLevel:   "error",
			expectContent: "record not found",
		},
		{
			name:      "Silent Level",
			logLevel:  gLogger.Silent,
			err:       errors.New("some error"),
			expectLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Truncate(testLogFile, 0)

			l := New(
				WithSlowThreshold(tt.slowThreshold),
				WithIgnoreRecordNotFoundError(tt.ignoreRNF),
			)
			l = l.LogMode(tt.logLevel)

			// Simulate elapsed time by adjusting begin time
			begin := now.Add(-tt.elapsed)
			l.Trace(ctx, begin, func() (string, int64) {
				return tt.sql, tt.rows
			}, tt.err)

			time.Sleep(10 * time.Millisecond)
			logContent := readAndClearLog()

			if tt.expectLog {
				assert.NotEmpty(t, logContent)
				assert.Contains(t, strings.ToLower(logContent), tt.expectLevel)
				assert.Contains(t, logContent, tt.expectContent)
				assert.Contains(t, logContent, "rows:")
			} else {
				assert.Empty(t, logContent)
			}
		})
	}
}

func TestInfoWarnError(t *testing.T) {
	l := New()
	l = l.LogMode(gLogger.Info)
	ctx := context.Background()

	// Test Info
	os.Truncate(testLogFile, 0)
	l.Info(ctx, "info message")
	time.Sleep(10 * time.Millisecond)
	content := readAndClearLog()
	assert.Contains(t, content, "info message")
	assert.Contains(t, strings.ToLower(content), "info")

	// Test Warn
	os.Truncate(testLogFile, 0)
	l.Warn(ctx, "warn message")
	time.Sleep(10 * time.Millisecond)
	content = readAndClearLog()
	assert.Contains(t, content, "warn message")
	assert.Contains(t, strings.ToLower(content), "warn")

	// Test Error
	os.Truncate(testLogFile, 0)
	l.Error(ctx, "error message")
	time.Sleep(10 * time.Millisecond)
	content = readAndClearLog()
	assert.Contains(t, content, "error message")
	assert.Contains(t, strings.ToLower(content), "error")
}

func TestSourceLocation(t *testing.T) {
	// Clean log before starting
	readAndClearLog()

	l := New()
	l = l.LogMode(gLogger.Info)
	ctx := context.Background()

	// Call Trace.
	l.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	// Give it a moment to flush
	time.Sleep(10 * time.Millisecond)

	logContent := readAndClearLog()
	t.Logf("Log content: %s", logContent)

	// We expect "logger_test.go" to appear in the log because the call is coming from this file.
	// We expect NOT to see "logger.go" as the source (which would mean it picked up the library file).
	assert.Contains(t, logContent, "logger_test.go", "Should contain the caller filename")
	assert.NotContains(t, logContent, "gormlogger/logger.go", "Should NOT contain the logger library filename as source")
}
