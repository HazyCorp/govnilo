package raterunner

import (
	"container/list"
	"sync"
)

type Percentile struct {
	mu     sync.Mutex
	vals   *list.List
	window uint64
	p      float64
}

func NewPercentile(p float64, window uint64) *Percentile {
	if p < 0 || p > 1 {
		panic("cannot calculate percentile. p must be in range 0 <= p <= 1")
	}

	return &Percentile{
		vals:   list.New(),
		window: window,
		p:      p,
	}
}

func (p *Percentile) Append(val uint64) {
	panic("implement me")
}

func (p *Percentile) GetStat() (float64, error) {
	panic("implement me")
}
