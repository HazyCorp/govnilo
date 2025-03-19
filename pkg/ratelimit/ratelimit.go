package ratelimit

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/HazyCorp/govnilo/common/hzlog"
	"golang.org/x/time/rate"
)

type Spec struct {
	Times uint64
	Per   time.Duration
}

var ErrStopped = errors.New("limiter stopped")

// need only to distinguish between parent error and child error
var internalStopError = errors.New("internal stop")

// Limiter is precise rate limiter with context support.
// Limiter guarantees, that on every window of defined size, there will be no more than
// requested amount of times.
type Limiter struct {
	l            *slog.Logger
	limiter      *rate.Limiter
	spec         Spec
	specRevision int

	mu   sync.Mutex
	cond *sync.Cond
}

// NewLimiter returns limiter that throttles rate of successful Acquire() calls
// to maxSize events at any given interval.
func New(spec Spec, opts ...RatelimitOption) *Limiter {
	var o ratelimitOpts
	for _, opt := range opts {
		opt.apply(&o)
	}

	l := o.l
	if l == nil {
		l = hzlog.NopLogger()
	}

	seconds := spec.Per.Seconds()

	var limit rate.Limit
	var burst = 1
	if spec.Times == 0 {
		limit = rate.Limit(0)
		burst = 0
	} else if seconds == 0 {
		limit = rate.Inf
	} else {
		limit = rate.Limit(float64(spec.Times) / seconds)
	}
	limiter := rate.NewLimiter(limit, burst)

	lim := &Limiter{
		l:            l,
		limiter:      limiter,
		spec:         spec,
		specRevision: 0,
	}
	cond := sync.NewCond(&lim.mu)
	lim.cond = cond

	return lim
}

func (l *Limiter) SetSpec(spec Spec) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if spec == l.spec {
		// don't need to change the spec
		return nil
	}

	seconds := spec.Per.Seconds()

	var limit rate.Limit
	var burst = 1
	if spec.Times == 0 {
		limit = rate.Limit(0)
		burst = 0
	} else if seconds == 0 {
		limit = rate.Inf
	} else {
		limit = rate.Limit(float64(spec.Times) / seconds)
	}

	l.limiter.SetLimit(limit)
	l.limiter.SetBurst(burst)
	l.spec = spec
	l.specRevision++

	// spec changed
	l.cond.Broadcast()

	return nil
}

func (l *Limiter) Acquire(ctx context.Context) error {
	// TODO: optimize/rewrite ourselves
	for {
		newCtx, cancel := context.WithCancelCause(ctx)
		waitEnded := false

		go func() {
			<-newCtx.Done()

			l.mu.Lock()
			waitEnded = true
			l.mu.Unlock()

			l.cond.Broadcast()
		}()

		l.mu.Lock()
		for l.limiter.Limit() == 0 && !waitEnded {
			l.cond.Wait()
		}

		// nonzero limit awaited, or context canceled

		if waitEnded {
			cancel(nil)
			l.mu.Unlock()
			return newCtx.Err()
		}
		l.mu.Unlock()

		go func() {
			l.mu.Lock()
			defer l.mu.Unlock()

			revision := l.specRevision
			for revision == l.specRevision && !waitEnded {
				l.cond.Wait()
			}

			if waitEnded {
				return
			}

			// spec changed, need to notify waiter
			cancel(internalStopError)
		}()

		err := l.limiter.Wait(newCtx)
		if err == nil {
			l.mu.Lock()
			// to avoid goroutine leak
			waitEnded = true
			l.mu.Unlock()

			// finally, awaited
			return nil
		}

		if newCtx.Err() == nil {
			l.l.DebugContext(ctx, "error from limiter, need to retry")
			continue
		}

		// newCtx.Err() is not nil, lets investigate it's cause
		err = context.Cause(newCtx)
		if errors.Is(err, internalStopError) {
			// it is a signal, that we should restart our waiting due to spec changed
			continue
		}

		l.mu.Lock()
		// to avoid goroutine leak
		waitEnded = true
		l.mu.Unlock()

		// context cancelled or deadline exceeded, returning an error to caller
		return newCtx.Err()
	}
}

func (l *Limiter) Stop(ctx context.Context) error {
	return nil
}
