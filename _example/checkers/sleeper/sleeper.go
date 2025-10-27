package usercreate

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/HazyCorp/govnilo/pkg/govnilo"

	"github.com/pkg/errors"
)

func init() {
	govnilo.RegisterChecker(NewSleeperChecker)
}

var _ govnilo.Checker = &SleeperChecker{}

type SleeperChecker struct {
	l *slog.Logger
	c *http.Client
}

func NewSleeperChecker(l *slog.Logger) *SleeperChecker {
	return &SleeperChecker{
		l: l,
		c: &http.Client{
			Timeout: time.Millisecond * 500,
		},
	}
}

func (c *SleeperChecker) Check(ctx context.Context, target string) ([]byte, error) {
	// Trace ID is automatically included in logs via hzlog context
	l := govnilo.GetLogger(ctx, c.l)
	l.DebugContext(ctx, "Starting CHECK operation")

	url := fmt.Sprintf("http://%s/operations/heavy", target)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, govnilo.InternalError(errors.Wrap(err, "cannot build the request"))
	}

	rsp, err := c.c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get response")
	}
	defer func() {
		io.Copy(io.Discard, rsp.Body)
		rsp.Body.Close()
	}()

	return nil, nil
}

func (c *SleeperChecker) Get(ctx context.Context, target string, data []byte) error {
	// Trace ID is automatically included in logs via hzlog context
	l := govnilo.GetLogger(ctx, c.l)
	l.DebugContext(ctx, "Starting GET operation")

	// always success. we don't need to check any consistency here
	return nil
}

func (c *SleeperChecker) CheckerID() govnilo.CheckerID {
	return govnilo.CheckerID{
		Service: "example",
		Name:    "sleeper",
	}
}
