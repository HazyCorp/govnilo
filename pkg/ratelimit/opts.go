package ratelimit

import "log/slog"

type ratelimitOpts struct {
	l *slog.Logger
}

type RatelimitOption interface {
	apply(o *ratelimitOpts)
}

type ratelimitOptionFunc func(o *ratelimitOpts)

func (f ratelimitOptionFunc) apply(o *ratelimitOpts) {
	f(o)
}

func WithLogger(l *slog.Logger) RatelimitOption {
	return ratelimitOptionFunc(func(o *ratelimitOpts) { o.l = l })
}
