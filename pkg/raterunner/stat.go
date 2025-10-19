package raterunner

type Stat interface {
	Append(val uint64)
	GetStat() (float64, error)
}