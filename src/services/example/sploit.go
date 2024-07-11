package example

import (
	"context"

	"go.uber.org/zap"

	"github.com/HazyCorp/checker/pkg/checker"
)

func init() {
	checker.RegisterSploit(NewExampleSploit)
}

// ExampleSploit implements checker.Sploit interface
var _ checker.Sploit = &ExampleSploit{}

type ExampleSploit struct {
	l    *zap.Logger
	prov checker.Provider
}

// Sploit constructor may return error as the last return value.

func NewExampleSploit(l *zap.Logger, prov checker.Provider) *ExampleSploit {
	return &ExampleSploit{l: l, prov: prov}
}

func (s *ExampleSploit) RunAttack(ctx context.Context, target string) error {
	s.l.Info("running attack on example service")
	return nil
}

func (s *ExampleSploit) SploitID() checker.SploitID {
	return checker.SploitID{Service: "example", Name: "drop_database_sql_injection"}
}
