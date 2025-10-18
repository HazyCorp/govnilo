package ratelimit

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
	"golang.org/x/time/rate"
)

type Spec struct {
	Times uint64
	Per   time.Duration
}

// Limiter is precise rate limiter with context support.
// Limiter guarantees, that on every window of defined size, there will be no more than
// requested amount of times.
type Limiter struct {
	l    *slog.Logger
	spec Spec

	rl *rate.Limiter

	mu       sync.Mutex
	nonNilRL *sync.Cond
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

	lim := Limiter{
		l:    l,
		spec: spec,
	}
	cond := sync.NewCond(&lim.mu)
	lim.nonNilRL = cond

	rl := lim.buildRL(spec)
	lim.rl = rl

	return &lim
}

func (l *Limiter) buildRL(spec Spec) *rate.Limiter {
	var rl *rate.Limiter
	if spec.Per == 0 {
		rl = rate.NewLimiter(rate.Inf, 1)
	} else if spec.Times != 0 {
		r := rate.Limit(float64(spec.Times) / spec.Per.Seconds())
		rl = rate.NewLimiter(r, 1)
	} else {
		rl = nil
	}

	return rl
}

func (l *Limiter) SetSpec(spec Spec) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.spec == spec {
		// nothing changed, dont need to notify anyone
		return nil
	}
	l.spec = spec

	newRL := l.buildRL(spec)
	l.rl = newRL
	if newRL != nil {
		l.nonNilRL.Broadcast()
	}

	return nil
}

func (l *Limiter) Acquire(ctx context.Context) error {
	// TODO: REWRITE THIS SHIT BLYAT!!!!
	// KAKIE NAHUI 2 GORUTINI NA OZHIDANIE?
	// DO EVERYTHING OURSELVES
	ctxDone := false
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	l.mu.Lock()
	if l.rl == nil {
		// start goroutine if and only if RL is nil now
		go func() {
			<-newCtx.Done()

			l.mu.Lock()
			ctxDone = true
			l.mu.Unlock()

			l.nonNilRL.Broadcast()
		}()
	}

	for l.rl == nil && !ctxDone {
		l.nonNilRL.Wait()
	}

	// we awaited for non nil rl, lets save it
	// (or rl is nil, but context done)
	rl := l.rl
	l.mu.Unlock()

	if ctxDone {
		return ctx.Err()
	}

	// avoid goroutines leak, stop awaiter
	cancel()

	// rl is non nil, we can wait
	withoutDeadline := ctxWithoutDeadline{Context: ctx}
	return rl.Wait(&withoutDeadline)
}

func (l *Limiter) Stop(ctx context.Context) error {
	return nil
}

type ctxWithoutDeadline struct {
	context.Context
}

func (c *ctxWithoutDeadline) Deadline() (time.Time, bool) {
	return time.Time{}, false
}
