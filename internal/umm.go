package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type TimeRange struct {
	Start time.Time
	End   *time.Time
}

type UMMErrors struct {
	Errors []string `json:"errors"`
	Err    error
}

func (e *UMMErrors) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return strings.Join(e.Errors, ";")
}

func newUMMError(r io.Reader) error {
	errs := &UMMErrors{}
	err := json.NewDecoder(r).Decode(&errs)
	if err != nil {
		errs.Err = fmt.Errorf("failed to decode error")
	}
	return errs
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
