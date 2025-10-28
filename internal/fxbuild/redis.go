package fxbuild

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/HazyCorp/govnilo/internal/configuration"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

type redisLogHook struct {
	l *slog.Logger
}

func (h *redisLogHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network string, addr string) (net.Conn, error) {
		h.l.InfoContext(
			ctx,
			"trying to dial with redis",
			slog.String("network", network),
			slog.String("addr", addr),
		)

		start := time.Now()
		conn, err := next(ctx, network, addr)
		if err != nil {
			h.l.ErrorContext(
				ctx,
				"failed to dial with redis",
				slog.String("network", network),
				slog.String("addr", addr),
				hzlog.Error(err),
			)
			return nil, err
		}

		h.l.InfoContext(
			ctx,
			"successfully dialed with redis",
			slog.String("network", network),
			slog.String("addr", addr),
			slog.Duration("elapsed", time.Since(start)),
		)

		return conn, nil
	}
}

func (h *redisLogHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.l.InfoContext(
			ctx,
			"trying to run command in redis",
			slog.String("cmd", cmd.String()),
		)

		start := time.Now()
		err := next(ctx, cmd)
		if err != nil {
			h.l.ErrorContext(
				ctx,
				"failed to run command in redis",
				slog.String("cmd", cmd.String()),
				hzlog.Error(err),
			)
			return err
		}

		h.l.InfoContext(
			ctx,
			"command in redis finished",
			slog.String("cmd", cmd.String()),
			slog.Duration("elapsed", time.Since(start)),
		)

		return nil
	}
}

func (h *redisLogHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		h.l.InfoContext(
			ctx,
			"trying to run pipeline in redis",
		)

		start := time.Now()
		err := next(ctx, cmds)
		if err != nil {
			h.l.ErrorContext(
				ctx,
				"failed to run pipeline in redis",
				hzlog.Error(err),
			)
			return err
		}

		h.l.InfoContext(
			ctx,
			"pipeline in redis finished",
			slog.Duration("elapsed", time.Since(start)),
		)

		return nil
	}
}
func NewRedisClient(c configuration.Redis, lc fx.Lifecycle, l *slog.Logger) *redis.Client {
	hook := &redisLogHook{l: l.With(slog.String("component", "infra:redis"))}
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Username: c.Username,
		Password: c.Password,
		DB:       c.DB,
	})
	client.AddHook(hook)

	lc.Append(fx.StartStopHook(
		func(ctx context.Context) error {
			if err := client.Ping(ctx).Err(); err != nil {
				return errors.Wrap(err, "cannot connect to redis")
			}

			return nil
		},
		func(ctx context.Context) {
			_ = client.Close()
		}))

	return client
}
