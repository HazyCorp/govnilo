package raterunner

import (
	"container/list"
	"slices"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Percentile struct {
	mu           sync.Mutex
	observations *list.List
	window       time.Duration
	p            float64
}

// TODO: use heap, if some latency problems will be present

type observation struct {
	val       uint64
	timestamp time.Time
}

func NewPercentile(p float64, window time.Duration) *Percentile {
	if p < 0 || p > 1 {
		panic("cannot calculate percentile. p must be in range 0 <= p <= 1")
	}

	return &Percentile{
		observations: list.New(),
		window:       window,
		p:            p,
	}
}

func (p *Percentile) Append(val uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	t := time.Now()
	o := observation{val: val, timestamp: t}

	p.observations.PushBack(o)
	p.cleanOutdatedLocked(t)

}

// TODO: use sync.Pool for slices (if performance matters)

func (p *Percentile) GetStat() (float64, error) {
	vals := make([]uint64, 0)

	func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		// delete outdated observations
		p.cleanOutdatedLocked(time.Now())

		// collect all the observations
		for v := p.observations.Front(); v != p.observations.Back(); v = v.Next() {
			vals = append(vals, v.Value.(observation).val)
		}
	}()

	if len(vals) == 0 {
		return 0, errors.Errorf("cannot calculate percentile for empty set of values")
	}

	// we need only snapshot here, don't need to hold mu for sort
	slices.Sort(vals)

	idx := int(p.p * float64(len(vals)))

	return float64(vals[idx]), nil
}

func (p *Percentile) cleanOutdatedLocked(now time.Time) {
	for p.observations.Len() > 0 {
		front := p.observations.Front()
		leftmostTimestamp := front.Value.(observation).timestamp
		if now.Sub(leftmostTimestamp) <= p.window {
			break
		}

		p.observations.Remove(front)
	}
}
