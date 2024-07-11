package example

import (
	"context"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/HazyCorp/checker/pkg/checker"
)

func init() {
	checker.RegisterChecker(NewExampleChecker)
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// ExampleChecker implements checker interface
var _ checker.Checker = &ExampleChecker{}

type ExampleChecker struct {
	l    *zap.Logger
	prov checker.Provider
}

// logger and provider are used as arguments of the constructor, it will be provided by the caller.
// Checker constructor may return error as the last return value.

func NewExampleChecker(l *zap.Logger, prov checker.Provider) *ExampleChecker {
	return &ExampleChecker{l: l, prov: prov}
}

func (e *ExampleChecker) Check(ctx context.Context, target string) ([]byte, error) {
	time.Sleep(time.Second)
	if rand.Intn(3) == 1 {
		return nil, errors.Errorf("unluck :(")
	}

	return []byte(`{"penis": "penis"}`), nil
}

func (e *ExampleChecker) Get(ctx context.Context, target string, data []byte) error {
	time.Sleep(time.Second)
	e.l.Info(string(data))

	return nil
}

func (e *ExampleChecker) ServiceName() string {
	return "example"
}
