package checkerctrl

import (
	"math/rand"
	"time"

	"github.com/pkg/errors"
)

type DummyStratedgy struct {
	SaveCapacity      uint64
	SaveProb          float64
	StoreDataDuration time.Duration
}

func NewDummySaveStratedgy(capacity uint64, saveProbability float64, storeDuration time.Duration) (*DummyStratedgy, error) {
	if saveProbability < 0 || saveProbability > 1 {
		return nil, errors.Errorf("probability must be between 0 and 1")
	}

	return &DummyStratedgy{
		SaveProb:          saveProbability,
		StoreDataDuration: storeDuration,
		SaveCapacity:      capacity,
	}, nil
}

func (s *DummyStratedgy) NeedSave(currentPoolSize uint64) bool {
	if s.SaveCapacity <= currentPoolSize {
		return false
	}

	return rand.Float64() < s.SaveProb
}

func (s *DummyStratedgy) NeedDelete(r *DataRecord) bool {
	// delete element if it is too old
	return time.Since(r.Created) > s.StoreDataDuration
}
