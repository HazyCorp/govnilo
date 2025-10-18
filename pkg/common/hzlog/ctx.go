package hzlog

import (
	"context"
	"log/slog"
)

type attrsKey struct{}

func ContextWith(ctx context.Context, attrs ...slog.Attr) context.Context {
	oldAttrs := getAttrs(ctx)

	newAttrs := make([]slog.Attr, 0, len(oldAttrs)+len(attrs))
	newAttrs = append(newAttrs, oldAttrs...)
	newAttrs = append(newAttrs, attrs...)

	return context.WithValue(ctx, attrsKey{}, newAttrs)
}

func getAttrs(ctx context.Context) []slog.Attr {
	currentAttrs := ctx.Value(attrsKey{})
	if currentAttrs == nil {
		return nil
	}

	// cannot panic
	return currentAttrs.([]slog.Attr)
}
