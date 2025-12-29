package requestid

import (
	"context"
	"github.com/google/uuid"
	"strings"
)

type ctxKey int

const keyRequestID ctxKey = 0

func Ctx(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, keyRequestID, genRequestID())
	return ctx
}

func Get(ctx context.Context) string {
	requestID, ok := ctx.Value(keyRequestID).(string)
	if !ok {
		return ""
	}
	return requestID
}

func genRequestID() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}
