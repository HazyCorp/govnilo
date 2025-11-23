package sleeper

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/HazyCorp/govnilo/pkg/govnilo"
	"go.opentelemetry.io/otel/trace"

	"github.com/pkg/errors"
)

var _ govnilo.Checker = &SleeperPutChecker{}

type SleeperPutChecker struct {
	l *slog.Logger
	c *http.Client
	s govnilo.RedisStorage[*SleeperEntity]
}

func NewSleeperPutChecker(l *slog.Logger, s *SleeperStorage) *SleeperPutChecker {
	return &SleeperPutChecker{
		l: l,
		c: &http.Client{
			Timeout: time.Millisecond * 500,
		},
		s: s,
	}
}

func (c *SleeperPutChecker) Check(ctx context.Context, target string) error {
	// Trace ID is automatically included in logs via hzlog context
	l := govnilo.GetLogger(ctx, c.l)
	l.DebugContext(ctx, "Starting CHECK operation")

	id := trace.SpanContextFromContext(ctx).TraceID().String()

	t := time.Now().String()
	e := &SleeperEntity{
		Id:    id,
		Value: t,
	}
	if err := c.s.Save(ctx, e); err != nil {
		return errors.Wrap(err, "cannot save sleeper entity")
	}

	return nil
}

func (c *SleeperPutChecker) CheckerID() govnilo.CheckerID {
	return govnilo.CheckerID{
		Service: "example",
		Name:    "sleeper-put",
	}
}
