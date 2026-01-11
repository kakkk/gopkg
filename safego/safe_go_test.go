package safego

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGo(t *testing.T) {
	ctx := context.Background()

	t.Run("normal", func(t *testing.T) {
		done := make(chan struct{})
		Go(ctx, func() {
			close(done)
		})
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for Go to finish")
		}
	})

	t.Run("panic", func(t *testing.T) {
		// Just ensure it doesn't crash the test process
		Go(ctx, func() {
			panic("test panic")
		})
		time.Sleep(100 * time.Millisecond)
	})
}

func TestGoFn(t *testing.T) {
	ctx := context.Background()

	t.Run("normal", func(t *testing.T) {
		fn := GoFn(ctx, func() error {
			return nil
		})
		err := fn()
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		testErr := errors.New("test error")
		fn := GoFn(ctx, func() error {
			return testErr
		})
		err := fn()
		if !errors.Is(err, testErr) {
			t.Errorf("expected error %v, got %v", testErr, err)
		}
	})

	t.Run("panic", func(t *testing.T) {
		fn := GoFn(ctx, func() error {
			panic("test panic")
		})
		err := fn()
		if err == nil {
			t.Fatal("expected non-nil error on panic recovery, got nil")
		}
		expected := "panic recovered: test panic"
		if err.Error() != expected {
			t.Errorf("expected error %q, got %q", expected, err.Error())
		}
	})
}
