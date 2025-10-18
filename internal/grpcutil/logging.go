package grpcutil

import (
	"context"
	"log/slog"
	"time"

	"github.com/HazyCorp/govnilo/pkg/common/hzlog"

	"google.golang.org/grpc"
)

func LoggingUnaryInterceptor(l *slog.Logger) grpc.UnaryServerInterceptor {
	return grpc.UnaryServerInterceptor(
		func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			start := time.Now()

			// TODO: use request ID or trace ID, and insert it in ctx
			r, err := handler(ctx, req)
			if err != nil {
				l.
					With(
						hzlog.Error(err),
						slog.String("method", info.FullMethod),
						slog.Duration("duration", time.Since(start)),
					).
					Error("error occurred while trying to handle request", slog.String("method", info.FullMethod))

				return r, err
			}

			l.
				With(
					slog.String("method", info.FullMethod),
					slog.Duration("duration", time.Since(start)),
				).
				Info("successfully handled request", slog.String("method", info.FullMethod))

			return r, nil
		},
	)
}
