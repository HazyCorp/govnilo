package util

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func CtxWithShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
			// pass (avoid goroutine leak)
		}
	}()

	return ctx, cancel
}
