package hzlog

import (
	"io"
	"log/slog"
)

var nopHandler = slog.NewTextHandler(io.Discard, nil)

func NopLogger() *slog.Logger {
	return slog.New(nopHandler)
}
