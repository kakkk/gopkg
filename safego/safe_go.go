package safego

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/kakkk/gopkg/logger"
)

func Go(ctx context.Context, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Ctx(ctx).Errorf("[safe.Go] panic recovered: %v, stack:\n%v", r, string(debug.Stack()))
			}
		}()
		fn()
	}()
}

func GoFn(ctx context.Context, fn func() error) func() error {
	return func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Ctx(ctx).Errorf("[safe.GoFn] panic recovered: %v, stack:\n%v", r, string(debug.Stack()))
				err = fmt.Errorf("panic recovered: %v", r)
			}
		}()
		return fn()
	}
}
