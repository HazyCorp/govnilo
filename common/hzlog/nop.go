package hzlog

import (
	"io"
	"log/slog"
)

func NopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
