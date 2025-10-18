package checkerctrl

type SaveStrategy interface {
	NeedSave(currentPoolSize uint64) bool
	NeedDelete(record *DataRecord) bool
}
