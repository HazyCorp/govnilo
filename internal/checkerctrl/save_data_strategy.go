package checkerctrl

type SaveStrategy interface {
	NeedSave(currentPoolSize uint64) bool
	NeedDelete(currentPool uint64) []uint64
}
