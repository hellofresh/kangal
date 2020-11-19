package interceptor

import (
	"context"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
)

const requestIDMDKey = "x-request-id"

// Logger is an interceptor that injects logger with request ID into the context of each RPC call.
type Logger struct {
	logger *zap.Logger
}

// NewLogger creates new Logger interceptor instance
func NewLogger(logger *zap.Logger) *Logger {
	return &Logger{logger: logger}
}

// Interceptor is the unary interceptor handler
func (m *Logger) Interceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	callID := uuid.Must(uuid.NewV4()).String()
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		requestID := md[requestIDMDKey]
		if len(requestID) > 0 {
			callID = requestID[0]
		}
	}

	ctx = mPkg.SetID(ctx, callID)

	callLogger := m.logger.With(zap.String("request-id", callID))
	ctx = mPkg.SetLogger(ctx, callLogger)

	return handler(ctx, req)
}
