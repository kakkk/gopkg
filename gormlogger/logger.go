package gormlogger

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"

	"github.com/kakkk/gopkg/logger"
)

type gormLogger struct {
	cfg *config
}

func New(opts ...Option) gLogger.Interface {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &gormLogger{
		cfg: cfg,
	}
}

func (l *gormLogger) LogMode(level gLogger.LogLevel) gLogger.Interface {
	newCfg := *l.cfg
	newCfg.logLevel = level
	return &gormLogger{
		cfg: &newCfg,
	}
}

func (l *gormLogger) Info(ctx context.Context, s string, args ...interface{}) {
	if l.cfg.logLevel >= gLogger.Info {
		logger.Ctx(ctx).Infof(s, args...)
	}
}

func (l *gormLogger) Warn(ctx context.Context, s string, args ...interface{}) {
	if l.cfg.logLevel >= gLogger.Warn {
		logger.Ctx(ctx).Warnf(s, args...)
	}
}

func (l *gormLogger) Error(ctx context.Context, s string, args ...interface{}) {
	if l.cfg.logLevel >= gLogger.Error {
		logger.Ctx(ctx).Errorf(s, args...)
	}
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.cfg.logLevel <= gLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	src := fileWithLineNum()

	if err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.cfg.ignoreRecordNotFoundError) {
		if l.cfg.logLevel >= gLogger.Error {
			logger.Ctx(ctx).WithError(err).Errorf("%s %s [%s] [rows:%d]", src, sql, elapsed, rows)
		}
		return
	}

	if l.cfg.slowThreshold != 0 && elapsed > l.cfg.slowThreshold && l.cfg.logLevel >= gLogger.Warn {
		logger.Ctx(ctx).Warnf("%s %s [%s] [rows:%d]", src, sql, elapsed, rows)
		return
	}

	if l.cfg.logLevel == gLogger.Info {
		logger.Ctx(ctx).Infof("%s %s [%s] [rows:%d]", src, sql, elapsed, rows)
	}
}

func fileWithLineNum() string {
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!strings.Contains(file, "gorm.io/gorm") && !strings.Contains(file, "gormlogger/logger.go")) {
			return fmt.Sprintf("%s:%d", file, line)
		}
	}
	return ""
}
