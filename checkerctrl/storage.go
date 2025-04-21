package checkerctrl

import (
	"context"
	"time"

	"github.com/HazyCorp/govnilo/hazycheck"
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type CheckerPointsStats struct {
	SuccessPoints       uint64
	PenaltyPoints       uint64
	TotalAttempts       int
	SuccessfullAttempts int
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type DataRecord struct {
	Data    []byte
	Created time.Time
}

type NeedDeleteFunc func(record *DataRecord) bool

type ControllerStorage interface {
	AppendCheckerData(ctx context.Context, checkerID hazycheck.CheckerID, data []byte) error
	GetCheckerDataPool(ctx context.Context, checkerID hazycheck.CheckerID) ([]DataRecord, error)
	RemoveDataFromPool(
		ctx context.Context,
		checkerID hazycheck.CheckerID,
		needDelete NeedDeleteFunc,
	) error

	Flush(ctx context.Context) error
}
