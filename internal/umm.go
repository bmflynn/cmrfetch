package internal

import (
	"fmt"
	"strings"
	"time"
)

type TimeRange struct {
	Start time.Time
	End   *time.Time
}

type CMRError struct {
	RequestID string
	Errors    []string `json:"errors"`
	Err       error
}

func (e *CMRError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf(
		"%s; request-id=%s", strings.Join(e.Errors, "; "), e.RequestID)
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
