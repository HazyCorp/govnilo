package sleeper

import (
	"context"
	"log/slog"

	"github.com/HazyCorp/govnilo/pkg/govnilo"

	"github.com/pkg/errors"
)

var _ govnilo.Checker = &SleeperPutChecker{}

type SleeperGetChecker struct {
	l *slog.Logger
	s govnilo.RedisStorage[*SleeperEntity]
}

func NewSleeperGetChecker(l *slog.Logger, s *SleeperStorage) *SleeperGetChecker {
	return &SleeperGetChecker{
		l: l,
		s: s,
	}
}

func (c *SleeperGetChecker) Check(ctx context.Context, target string) error {
	// Trace ID is automatically included in logs via hzlog context
	l := govnilo.GetLogger(ctx, c.l)
	l.DebugContext(ctx, "starting operation")

	sleeper, err := c.s.GetRandom(ctx)
	if err != nil {
		if errors.Is(err, govnilo.ErrNotFound) {
			return govnilo.InternalError(errors.Wrap(err, "no sleeper entity found"))
		}
		return errors.Wrap(err, "cannot get sleeper entity")
	}

	l.DebugContext(ctx, "sleeper entity retrieved", slog.Any("sleeper", sleeper))

	return nil
}

func (c *SleeperGetChecker) CheckerID() govnilo.CheckerID {
	return govnilo.CheckerID{
		Service: "example",
		Name:    "sleeper-retrieve",
	}
}
