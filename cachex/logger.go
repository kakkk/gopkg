package cachex

import (
	"context"
	"fmt"
	"log/slog"
)

// defaultLogger 默认logger，使用标准库 log/slog
type defaultLogger struct{}

func newDefaultLogger() Logger {
	return defaultLogger{}
}

func (d defaultLogger) Infof(ctx context.Context, format string, a ...any) {
	slog.Info(fmt.Sprintf(format, a...))
}

func (d defaultLogger) Warnf(ctx context.Context, format string, a ...any) {
	slog.Warn(fmt.Sprintf(format, a...))
}

func (d defaultLogger) Errorf(ctx context.Context, format string, a ...any) {
	slog.Error(fmt.Sprintf(format, a...))
}
