package raterunner

import (
	"container/list"
	"sync"

	"github.com/pkg/errors"
)

const DefaultWindowSize = 128

type AvgCounterSpec struct {
	WindowSize uint64
}

type AvgCounter struct {
	spec AvgCounterSpec
	mu   sync.Mutex
	vals *list.List
	sum  uint64
}

func NewAvgCounter(spec AvgCounterSpec) *AvgCounter {
	return &AvgCounter{spec: spec, vals: list.New()}
}

func (c *AvgCounter) Append(val uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for c.vals.Len() >= int(c.spec.WindowSize) {
		front := c.vals.Front()
		v := front.Value.(uint64)
		c.sum -= v
		c.vals.Remove(front)
	}

	c.vals.PushBack(val)
	c.sum += val
}

func (c *AvgCounter) GetAvg() (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.vals.Len() == 0 {
		return 0, errors.Errorf("list of values is empty, cannot calculate average of no values")
	}

	return float64(c.sum) / float64(c.vals.Len()), nil
}
