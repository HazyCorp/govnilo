package checkerctrl

import (
	"context"
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

type ControllerStorage interface {
	AppendCheckerData(ctx context.Context, checkerID hazycheck.CheckerID, data []byte) error
	GetCheckerDataPool(ctx context.Context, checkerID hazycheck.CheckerID) ([][]byte, error)
	RemoveDataFromPool(
		ctx context.Context,
		checkerID hazycheck.CheckerID,
		idx uint64,
	) ([]byte, error)

	Flush(ctx context.Context) error
}
