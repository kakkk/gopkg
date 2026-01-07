package hertzlogger

import (
	"context"
	"io"

	"github.com/cloudwego/hertz/pkg/common/hlog"

	"github.com/kakkk/gopkg/logger"
)

type hertzLogger struct{}

func New() hlog.FullLogger {
	return &hertzLogger{}
}

func (l *hertzLogger) Trace(v ...interface{}) {
	logger.Trace(v...)
}

func (l *hertzLogger) Debug(v ...interface{}) {
	logger.Debug(v...)
}

func (l *hertzLogger) Info(v ...interface{}) {
	logger.Info(v...)
}

func (l *hertzLogger) Notice(v ...interface{}) {
	logger.Warn(v...)
}

func (l *hertzLogger) Warn(v ...interface{}) {
	logger.Warn(v...)
}

func (l *hertzLogger) Error(v ...interface{}) {
	logger.Error(v...)
}

func (l *hertzLogger) Fatal(v ...interface{}) {
	logger.Fatal(v...)
}

func (l *hertzLogger) Tracef(format string, v ...interface{}) {
	logger.Tracef(format, v...)
}

func (l *hertzLogger) Debugf(format string, v ...interface{}) {
	logger.Debugf(format, v...)
}

func (l *hertzLogger) Infof(format string, v ...interface{}) {
	logger.Infof(format, v...)
}

func (l *hertzLogger) Noticef(format string, v ...interface{}) {
	logger.Warnf(format, v...)
}

func (l *hertzLogger) Warnf(format string, v ...interface{}) {
	logger.Warnf(format, v...)
}

func (l *hertzLogger) Errorf(format string, v ...interface{}) {
	logger.Errorf(format, v...)
}

func (l *hertzLogger) Fatalf(format string, v ...interface{}) {
	logger.Fatalf(format, v...)
}

func (l *hertzLogger) CtxTracef(ctx context.Context, format string, v ...interface{}) {
	logger.WithContext(ctx).Tracef(format, v...)
}

func (l *hertzLogger) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	logger.WithContext(ctx).Debugf(format, v...)
}

func (l *hertzLogger) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	logger.WithContext(ctx).Infof(format, v...)
}

func (l *hertzLogger) CtxNoticef(ctx context.Context, format string, v ...interface{}) {
	logger.WithContext(ctx).Warnf(format, v...)
}

func (l *hertzLogger) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	logger.WithContext(ctx).Warnf(format, v...)
}

func (l *hertzLogger) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	logger.WithContext(ctx).Errorf(format, v...)
}

func (l *hertzLogger) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	logger.WithContext(ctx).Fatalf(format, v...)
}

func (l *hertzLogger) SetLevel(level hlog.Level) {
	// No-op: The logging level is managed globally by the underlying logger package.
	// Hertz-specific level settings are ignored to maintain consistency across the application.
	return
}

func (l *hertzLogger) SetOutput(writer io.Writer) {
	// No-op: The logging output is managed globally by the underlying logger package.
	// Hertz-specific output settings are ignored to maintain consistency across the application.
	return
}
