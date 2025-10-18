package example

import (
	"context"
	"log/slog"
	"time"

	"github.com/HazyCorp/govnilo/pkg/govnilo"
)

func init() {
	govnilo.RegisterSploit(NewExampleSploit)
}

var _ govnilo.Sploit = &ExampleSploit{}

type ExampleSploit struct {
	l *slog.Logger
}

func NewExampleSploit(l *slog.Logger) *ExampleSploit {
	return &ExampleSploit{
		l: l,
	}
}

func (s *ExampleSploit) RunAttack(ctx context.Context, target string) error {
	// Trace ID is automatically included in logs via hzlog context
	l := govnilo.GetLogger(ctx, s.l)
	l.Debug("Starting RunAttack operation")

	// Simulate some attack operation
	l.Info("Running attack against target", slog.String("target", target))

	// Simulate attack duration
	time.Sleep(100 * time.Millisecond)

	// Trace ID persists throughout the operation
	l.Debug("Attack completed")

	return nil
}

func (s *ExampleSploit) SploitID() govnilo.SploitID {
	return govnilo.SploitID{
		Service: "example",
		Name:    "example_attack",
	}
}
