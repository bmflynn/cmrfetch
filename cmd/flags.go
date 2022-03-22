package cmd

import (
	"fmt"
	"strings"
	"time"
)

type TimeVal time.Time

func (t *TimeVal) String() string { return "" }
func (t *TimeVal) Type() string   { return "timestamp" }
func (t *TimeVal) Set(s string) error {
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	*t = TimeVal(x)
	return nil
}

type TimerangeVal []time.Time

func (tr *TimerangeVal) String() string { return "" }
func (tr *TimerangeVal) Type() string   { return "timerange" }
func (tr *TimerangeVal) Set(s string) error {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return fmt.Errorf("expected 2 componets, got %d", len(parts))
	}

	x := TimerangeVal([]time.Time{})

	t1, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return fmt.Errorf("invalid start time")
	}
	x = append(x, t1)
	t2, err := time.Parse(time.RFC3339, parts[1])
	if err != nil {
		return fmt.Errorf("invalid end time")
	}
	x = append(x, t2)
	*tr = x
	return nil
}
