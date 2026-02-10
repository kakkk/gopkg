package logger_test

import (
	"context"
	"testing"

	"github.com/kakkk/gopkg/logger"
)

func TestCaller(t *testing.T) {

	logger.Init(logger.WithLineNumber(true))

	logger.Info("test caller info")

	logger.Infof("test caller infof: %s", "hello")

	ctx := context.Background()

	logger.Ctx(ctx).Info("test caller ctx info")

}
