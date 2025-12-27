package dlock

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func validateKeyAndTTL(key string, ttl time.Duration) error {
	if strings.TrimSpace(key) == "" {
		return ErrInvalidKey
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	return nil
}

func lockValue() string {
	return uuid.New().String()
}
