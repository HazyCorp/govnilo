package fxutil

import "go.uber.org/fx"

func AsIface[TIface any](constructor interface{}) interface{} {
	return fx.Annotate(constructor, fx.As(new(TIface)))
}

func Need[T any](val T) {}
