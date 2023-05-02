package internal

import (
	"fmt"
	"time"
)

type TimeRange struct {
	Start time.Time
	End   *time.Time
}

type CMRError struct {
	RequestID string
	Status    string
	Err       error
}

func (e *CMRError) Error() string {
	return fmt.Sprintf("%s; error=%s; request-id=%s", e.Status, e.Err, e.RequestID)
}

func encodeTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05Z")
}

func encodeTimeRange(tr TimeRange) string {
	s := encodeTime(tr.Start) + ","
	if tr.End != nil {
		s += encodeTime(*tr.End)
	}
	return s
}
