package checker

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/HazyCorp/govnilo/govnilo"
)

func init() {
	govnilo.RegisterChecker(New)
}

type ExampleChecker struct{}

func New() *ExampleChecker {
	return &ExampleChecker{}
}

func (c *ExampleChecker) Check(ctx context.Context, target string) ([]byte, error) {
	time.Sleep(time.Second)

	if rand.Intn(5) == 0 {
		return []byte("unluck"), errors.New("unluck :(")
	}

	return []byte("lucky day"), nil
}

func (c *ExampleChecker) Get(ctx context.Context, target string, data []byte) error {
	time.Sleep(time.Second * 2)
	return nil
}

func (c *ExampleChecker) CheckerID() govnilo.CheckerID {
	return govnilo.CheckerID{
		Service: "example",
		Name:    "random",
	}
}
