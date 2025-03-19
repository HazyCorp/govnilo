package ratelimit

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/time/rate"
)

type RateLimitterSpec struct {
	Times uint64
	Per   time.Duration
}

type waiter struct {
	ch chan struct{}
}

// RateLimitter is precise rate limiter with context support.
// RateLimitter guarantees, that on every window of defined size, there will be no more than
// requested amount of times.
type RateLimitter struct {
	limiter *rate.Limiter
}

var ErrStopped = errors.New("limiter stopped")

// NewLimiter returns limiter that throttles rate of successful Acquire() calls
// to maxSize events at any given interval.
func New(l *slog.Logger, spec RateLimitterSpec) *RateLimitter {
	seconds := spec.Per.Seconds()

	var limit rate.Limit
	if seconds == 0 {
		limit = rate.Inf
	} else {
		limit = rate.Limit(float64(spec.Times) / seconds)
	}
	limiter := rate.NewLimiter(limit, 1)

	return &RateLimitter{limiter: limiter}
}

func (l *RateLimitter) SetSpec(spec RateLimitterSpec) error {
	seconds := spec.Per.Seconds()

	var limit rate.Limit
	if seconds == 0 {
		limit = rate.Inf
	} else {
		limit = rate.Limit(float64(spec.Times) / seconds)
	}
	l.limiter.SetLimit(limit)

	return nil
}

func (l *RateLimitter) Acquire(ctx context.Context) error {
	return l.limiter.Wait(ctx)
}

func (l *RateLimitter) Stop(ctx context.Context) error {
	return nil
}
