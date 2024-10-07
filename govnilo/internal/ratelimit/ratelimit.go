package ratelimit

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
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
	l *zap.Logger

	spec    RateLimitterSpec
	stopped chan struct{}

	mu                               *sync.Mutex
	daemonsWg                        *sync.WaitGroup
	waitersNotEmptyAndSpaceAvailable *sync.Cond
	waiters                          *list.List
	acquired                         *list.List
	acquiredNotEmpty                 *sync.Cond
}

var ErrStopped = errors.New("limiter stopped")

// NewLimiter returns limiter that throttles rate of successful Acquire() calls
// to maxSize events at any given interval.
func New(l *zap.Logger, spec RateLimitterSpec) *RateLimitter {
	var mu sync.Mutex
	spaceAvailable := sync.NewCond(&mu)
	acquiredNotEmpty := sync.NewCond(&mu)

	ret := &RateLimitter{
		l: l,

		spec:    spec,
		stopped: make(chan struct{}),

		mu:                               &mu,
		daemonsWg:                        &sync.WaitGroup{},
		waitersNotEmptyAndSpaceAvailable: spaceAvailable,
		waiters:                          list.New(),
		acquired:                         list.New(),
		acquiredNotEmpty:                 acquiredNotEmpty,
	}

	ret.daemonsWg.Add(2)
	go ret.startPullingWaiters()
	go ret.startNotifierOnAvailableSpace()

	return ret
}

func (l *RateLimitter) SetSpec(spec RateLimitterSpec) error {
	l.mu.Lock()
	l.spec = spec
	l.mu.Unlock()

	l.acquiredNotEmpty.Broadcast()
	l.waitersNotEmptyAndSpaceAvailable.Broadcast()

	return nil
}

func (l *RateLimitter) Acquire(ctx context.Context) error {
	if l.closed() {
		return ErrStopped
	}

	l.mu.Lock()
	if l.spec.Per == time.Duration(0) {
		// rate limit disabled
		l.mu.Unlock()
		return nil
	}

	if l.acquired.Len() < int(l.spec.Times) {
		// if there are some space for acquiring, acquire it without any waiting
		l.acquired.PushBack(time.Now().Add(l.spec.Per))
		l.mu.Unlock()
		l.acquiredNotEmpty.Broadcast()

		// successfully acquired limiter as a fast path
		return nil
	}

	awaiter := waiter{ch: make(chan struct{})}
	element := l.waiters.PushBack(&awaiter)

	l.mu.Unlock()
	l.waitersNotEmptyAndSpaceAvailable.Broadcast()

	select {
	case <-awaiter.ch:
		// awaited signal about adding our time to acquired list
		return nil

	case <-ctx.Done():
		// need to delete us from awaiter list
		l.mu.Lock()
		l.waiters.Remove(element)
		l.mu.Unlock()

		return ctx.Err()

	case <-l.stopped:
		// shutdown
		return ErrStopped
	}
}

func (l *RateLimitter) Stop(ctx context.Context) error {
	close(l.stopped)

	// notify every routine about stop
	l.waitersNotEmptyAndSpaceAvailable.Broadcast()
	l.acquiredNotEmpty.Broadcast()

	wait := make(chan struct{}, 1)
	go func() { l.daemonsWg.Wait(); wait <- struct{}{} }()

	select {
	case <-wait:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *RateLimitter) startPullingWaiters() {
	defer l.daemonsWg.Done()

	l.mu.Lock()
	defer l.mu.Unlock()

	for {
		spec := l.spec

		for (l.acquired.Len() == int(spec.Times) || l.waiters.Len() == 0) && !l.closed() && spec == l.spec {
			l.waitersNotEmptyAndSpaceAvailable.Wait()
		}

		if l.closed() {
			return
		}
		if spec != l.spec {
			// spec changed, need to restart iteration
			continue
		}

		for l.acquired.Len() < int(spec.Times) && l.waiters.Len() > 0 {
			el := l.waiters.Front()
			l.waiters.Remove(el)

			w := el.Value.(*waiter)
			l.acquired.PushBack(time.Now().Add(spec.Per))

			close(w.ch)
		}

		l.acquiredNotEmpty.Broadcast()
	}
}

func (l *RateLimitter) startNotifierOnAvailableSpace() {
	defer l.daemonsWg.Done()

	// create reusable timer, need to drain before usage
	timer := time.NewTimer(0 * time.Second)
	if !timer.Stop() {
		<-timer.C
	}

	l.mu.Lock()

	for {
		spec := l.spec

		for l.acquired.Len() == 0 && !l.closed() && spec == l.spec {
			l.acquiredNotEmpty.Wait()
		}

		if l.closed() {
			l.mu.Unlock()
			return
		}
		if l.spec != spec {
			continue
		}

		// acquired not empty after cond.Wait. waiting for the leftmost node expiration
		timer.Reset(time.Until(l.acquired.Front().Value.(time.Time)))

		// need to sleep without lock acquired
		l.mu.Unlock()

		select {
		case <-timer.C:
			// pass
		case <-l.stopped:
			if !timer.Stop() {
				<-timer.C
			}

			return
		}

		// all actions with data structures must be done under mutex
		l.mu.Lock()

		for l.acquired.Len() > 0 && l.acquired.Front().Value.(time.Time).Before(time.Now()) {
			l.acquired.Remove(l.acquired.Front())
		}

		if l.waiters.Len() > 0 && l.acquired.Len() < int(l.spec.Times) {
			l.waitersNotEmptyAndSpaceAvailable.Broadcast()
		}
	}
}

func (l *RateLimitter) closed() bool {
	select {
	case <-l.stopped:
		// pass
	default:
		return false
	}

	return true
}
