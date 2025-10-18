package statestore

import (
	"context"
	"log/slog"
)

type defaultFunc[TState any] func() TState

type StateStore[TState any] interface {
	RetrieveState(ctx context.Context) (TState, error)
	UpdateState(ctx context.Context, updateF func(s *TState) error) error
	Flush(ctx context.Context) error
}

type stateStoreOptions struct {
	logger *slog.Logger
}

type StateStoreOpt interface {
	apply(*stateStoreOptions)
}

type stateStoreOptFunc func(o *stateStoreOptions)

func (f stateStoreOptFunc) apply(o *stateStoreOptions) {
	f(o)
}

func WithLogger(l *slog.Logger) StateStoreOpt {
	return stateStoreOptFunc(func(o *stateStoreOptions) {
		o.logger = l
	})
}
