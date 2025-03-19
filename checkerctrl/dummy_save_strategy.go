package checkerctrl

import (
	"math/rand"

	"github.com/pkg/errors"
)

type DummyStratedgy struct {
	SaveCapacity uint64
	SaveProb     float64
}

func NewDummySaveStratedgy(capacity uint64, saveProbability float64) (*DummyStratedgy, error) {
	if saveProbability < 0 || saveProbability > 1 {
		return nil, errors.Errorf("probability must be between 0 and 1")
	}

	return &DummyStratedgy{
		SaveCapacity: capacity,
		SaveProb:     saveProbability,
	}, nil
}

func (s *DummyStratedgy) NeedSave(currentPoolSize uint64) bool {
	return rand.Float64() < s.SaveProb
}

func (s *DummyStratedgy) NeedDelete(currentPoolSize uint64) []uint64 {
	if s.SaveCapacity < currentPoolSize {
		return nil
	}

	indices := make([]uint64, 0)
	for currentPoolSize > s.SaveCapacity {
		indices = append(indices, uint64(rand.Int63n(int64(currentPoolSize))))
	}

	return indices
}
