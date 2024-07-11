package statestore

import "context"

type StateStore[TState any] interface {
	RetrieveState(ctx context.Context) (TState, error)
	UpdateState(ctx context.Context, updateF func(s *TState) error) error
}
