package interceptor

import (
	"context"
	"runtime/debug"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
)

// Recovery is an unary interceptor that recovers from panics, logs the panic (and a
// backtrace), and returns gRPC Internal error status if
// possible. Recoverer prints a request ID if one is provided.
type Recovery struct{}

// NewRecovery creates new Recovery interceptor instance
func NewRecovery() *Recovery {
	return &Recovery{}
}

// Interceptor is the unary interceptor handler
func (m *Recovery) Interceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger := mPkg.GetLogger(ctx)

			logger.Error(
				"Internal server error handled",
				zap.Any("error", r),
				zap.ByteString("trace", debug.Stack()),
			)

			err = status.Errorf(codes.Internal, "%s", r)
		}
	}()

	return handler(ctx, req)
}
