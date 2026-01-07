package hertzlogger

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/kakkk/gopkg/logger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const testLogFile = "hertz_test.log"

func TestMain(m *testing.M) {
	// Initialize logger to write to a file for testing
	// We must ensure this is the first time Init is called for this package's test binary
	logger.Init(
		logger.WithFileName(testLogFile),
		logger.WithConsoleOutput(false),
		logger.WithJSONFormat(false),
		logger.WithLevel(logrus.TraceLevel), // Capture all levels
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
	// Clear the file so next test starts fresh
	os.Truncate(testLogFile, 0)
	return string(content)
}

// Verify interface compliance
var _ hlog.FullLogger = (*hertzLogger)(nil)

func TestNew(t *testing.T) {
	l := New()
	assert.NotNil(t, l)
}

func TestHertzLogger_Levels(t *testing.T) {
	l := New()
	ctx := context.Background()

	tests := []struct {
		name          string
		logFunc       func()
		expectLevel   string // Expected string in log file (e.g. "info", "warning")
		expectMessage string
	}{
		{
			name: "Trace",
			logFunc: func() {
				l.Trace("trace message")
			},
			expectLevel:   "trace",
			expectMessage: "trace message",
		},
		{
			name: "Debug",
			logFunc: func() {
				l.Debug("debug message")
			},
			expectLevel:   "debug",
			expectMessage: "debug message",
		},
		{
			name: "Info",
			logFunc: func() {
				l.Info("info message")
			},
			expectLevel:   "info",
			expectMessage: "info message",
		},
		{
			name: "Notice -> Warn",
			logFunc: func() {
				l.Notice("notice message")
			},
			expectLevel:   "warn", // Hertz Notice maps to Warn
			expectMessage: "notice message",
		},
		{
			name: "Warn",
			logFunc: func() {
				l.Warn("warn message")
			},
			expectLevel:   "warn",
			expectMessage: "warn message",
		},
		{
			name: "Error",
			logFunc: func() {
				l.Error("error message")
			},
			expectLevel:   "error",
			expectMessage: "error message",
		},
		{
			name: "Tracef",
			logFunc: func() {
				l.Tracef("tracef %s", "msg")
			},
			expectLevel:   "trace",
			expectMessage: "tracef msg",
		},
		{
			name: "CtxInfof",
			logFunc: func() {
				l.CtxInfof(ctx, "ctx info %s", "msg")
			},
			expectLevel:   "info",
			expectMessage: "ctx info msg",
		},
		{
			name: "CtxNoticef -> Warn",
			logFunc: func() {
				l.CtxNoticef(ctx, "ctx notice %s", "msg")
			},
			expectLevel:   "warn",
			expectMessage: "ctx notice msg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear log before test
			readAndClearLog()

			// Run log function
			tt.logFunc()

			// Wait a bit for async I/O (though logger writes are usually sync or fast enough)
			time.Sleep(10 * time.Millisecond)

			// Check content
			content := readAndClearLog()
			assert.NotEmpty(t, content, "Log file should not be empty")
			assert.Contains(t, strings.ToLower(content), tt.expectLevel, "Log should contain correct level")
			assert.Contains(t, content, tt.expectMessage, "Log should contain message")
		})
	}
}

func TestHertzLogger_NoOps(t *testing.T) {
	// Ensure these don't panic or cause side effects that break other tests
	l := New()
	l.SetLevel(hlog.LevelDebug)
	l.SetOutput(nil)
}
