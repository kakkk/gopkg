package gormlogger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	assert.Equal(t, 2*time.Second, cfg.slowThreshold)
	assert.True(t, cfg.ignoreRecordNotFoundError)
	assert.Equal(t, logger.Warn, cfg.logLevel)
}

func TestWithSlowThreshold(t *testing.T) {
	cfg := defaultConfig()
	WithSlowThreshold(5 * time.Second)(cfg)
	assert.Equal(t, 5*time.Second, cfg.slowThreshold)
}

func TestWithIgnoreRecordNotFoundError(t *testing.T) {
	cfg := defaultConfig()
	WithIgnoreRecordNotFoundError(false)(cfg)
	assert.False(t, cfg.ignoreRecordNotFoundError)
}

func TestWithLogLevel(t *testing.T) {
	cfg := defaultConfig()
	WithLogLevel(logger.Info)(cfg)
	assert.Equal(t, logger.Info, cfg.logLevel)
}
