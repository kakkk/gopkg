package requestid

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type ctxKey int

const keyRequestID ctxKey = 0

func Ctx(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, keyRequestID, Gen())
	return ctx
}

func Get(ctx context.Context) string {
	requestID, ok := ctx.Value(keyRequestID).(string)
	if !ok {
		return ""
	}
	return requestID
}

func Gen() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}

func Set(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ctx, keyRequestID, requestID)
	return ctx
}
