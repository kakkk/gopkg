package dlock

import "errors"

// 错误定义
var (
	ErrLockNotAcquired = errors.New("lock not acquired")
	ErrLockAlreadyHeld = errors.New("lock already held")
	ErrLockNotHeld     = errors.New("lock not held")
	ErrInvalidTTL      = errors.New("invalid ttl")
	ErrInvalidKey      = errors.New("invalid lockKey")
)
