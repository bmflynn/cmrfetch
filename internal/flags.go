package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

var timeRangeLayouts = []string{
	time.RFC3339,
	"2006-01-02",
}

type TimeRangeValue struct {
	Start *time.Time
	End   *time.Time
}

func NewTimeRangeValue() TimeRangeValue {
	tr := TimeRangeValue{}
	tr.Set(tr.String())
	return tr
}

// String returns a default range of 24 hours ago to now.
func (v *TimeRangeValue) String() string {
	now := time.Now().Add(-24 * time.Hour).UTC()
	return now.Format(time.RFC3339) + ","
}

func (v *TimeRangeValue) parse(s string) *time.Time {
	for _, layout := range timeRangeLayouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return &t
		}
	}
	return nil
}

func (tr *TimeRangeValue) Set(val string) error {
	parts := strings.SplitN(val, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected <start>,[<end>], with format <yyyy-mm-dd>[T<hh:mm:ss>Z]")
	}

	x := &TimeRangeValue{}
	x.Start = x.parse(parts[0])
	if x.Start == nil {
		return fmt.Errorf("invalid start time format; expected <yyyy-mm-dd>[T<hh:mm:ss>Z]")
	}
	if parts[1] != "" {
		x.End = x.parse(parts[1])
		if x.End == nil {
			return fmt.Errorf("invalid end time format; expected <yyyy-mm-dd>[T<hh:mm:ss>Z]")
		}
	}
	*tr = *x
	return nil
}

func (v *TimeRangeValue) Type() string { return "timerange" }

var _ pflag.Value = (*TimeRangeValue)(nil)
