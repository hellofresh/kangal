package middleware

import (
	"context"

	"go.uber.org/zap"
)

type requestCtxKey int

const (
	requestIDKey requestCtxKey = iota
	requestLoggerKey
)

// SetID sets ID
func SetID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// SetLogger sets logger
func SetLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, requestLoggerKey, logger)
}

// GetLogger returns a request logger from the given context if one is present.
func GetLogger(ctx context.Context) *zap.Logger {
	if ctx == nil {
		panic("Can not get request logger from empty context")
	}
	if requestLogger, ok := ctx.Value(requestLoggerKey).(*zap.Logger); ok {
		return requestLogger
	}

	return nil
}

// GetID returns request ID
func GetID(ctx context.Context) string {
	if ctx == nil {
		panic("Can not get request ID from empty context")
	}

	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}

	return ""
}
