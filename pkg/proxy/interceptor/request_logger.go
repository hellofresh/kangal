package interceptor

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
)

// RequestLogger is an interceptor that logs API call stats
type RequestLogger struct {
}

// NewRequestLogger creates new RequestLogger interceptor instance
func NewRequestLogger() *RequestLogger {
	return &RequestLogger{}
}

// Interceptor is the unary interceptor handler
func (m *RequestLogger) Interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
	logger := mPkg.GetLogger(ctx)

	logger.Debug("Starting request", zap.String("method", info.FullMethod))

	startedAt := time.Now()

	response, err := handler(ctx, req)

	elapsed := time.Now().Sub(startedAt)
	requestStatus, _ := status.FromError(err)

	logger.Info(
		"Finished serving request",
		zap.String("method", info.FullMethod),
		zap.Int64("elapsed", elapsed.Milliseconds()),
		zap.String("elapsed-fmt", elapsed.String()),
		zap.String("status", requestStatus.Code().String()),
		zap.Error(requestStatus.Err()),
	)

	return response, err
}
