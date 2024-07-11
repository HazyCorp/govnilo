package grpcutil

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func LoggingUnaryInterceptor(l *zap.Logger) grpc.UnaryServerInterceptor {
	return grpc.UnaryServerInterceptor(
		func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			start := time.Now()

			// TODO: use request ID or trace ID, and insert it in ctx
			r, err := handler(ctx, req)
			if err != nil {
				l.
					With(
						zap.Error(err),
						zap.String("method", info.FullMethod),
						zap.Duration("duration", time.Since(start)),
					).
					Sugar().
					Errorf("error occurred while trying to handle %s", info.FullMethod)

				return r, err
			}

			l.
				With(
					zap.String("method", info.FullMethod),
					zap.Duration("duration", time.Since(start)),
				).
				Sugar().
				Infof("successfully handled %s method", info.FullMethod)

			return r, nil
		},
	)
}
