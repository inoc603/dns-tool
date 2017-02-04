package types

import "time"

type Report struct {
	QPS       float64
	QTotal    int64
	Duration  time.Duration
	RTTAvg    time.Duration
	RTTStdDev time.Duration
}
