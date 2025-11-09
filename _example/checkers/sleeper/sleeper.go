package usercreate

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/HazyCorp/govnilo/pkg/govnilo"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

func init() {
	govnilo.RegisterChecker(NewSleeperChecker)
}

var _ govnilo.Checker = &SleeperChecker{}

type SleeperChecker struct {
	l *slog.Logger
	c *http.Client
	r *redis.Client
}

func NewSleeperChecker(l *slog.Logger, r *redis.Client) *SleeperChecker {
	return &SleeperChecker{
		l: l,
		c: &http.Client{
			Timeout: time.Millisecond * 500,
		},
		r: r,
	}
}

func (c *SleeperChecker) Check(ctx context.Context, target string) error {
	// Trace ID is automatically included in logs via hzlog context
	l := govnilo.GetLogger(ctx, c.l)
	l.DebugContext(ctx, "Starting CHECK operation")

	t := time.Now().String()
	err := c.r.Set(ctx, "sleeper-check", t, redis.KeepTTL).Err()
	if err != nil {
		return errors.Wrap(err, "cannot set redis key")
	}
	l.DebugContext(ctx, "successfully set redis key", slog.String("key", "sleeper-check"), slog.String("value", t))

	return nil
}

func (c *SleeperChecker) CheckerID() govnilo.CheckerID {
	return govnilo.CheckerID{
		Service: "example",
		Name:    "sleeper",
	}
}
