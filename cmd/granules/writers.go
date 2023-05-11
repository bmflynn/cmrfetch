package granules

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bmflynn/cmrfetch/internal"
	"github.com/jedib0t/go-pretty/v6/table"
)

type outputWriter func(internal.GranuleResult, io.Writer, []string) error

func longWriter(zult internal.GranuleResult, w io.Writer, fields []string) error {
	return nil
}

func shortWriter(zult internal.GranuleResult, w io.Writer, _ []string) error {
	fields := []string{"name", "size", "native_id", "concept_id", "revision_id"}
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(table.StyleLight)
	header := table.Row{}
	for _, name := range fields {
		header = append(header, name)
	}
	t.AppendHeader(header)

	for granule := range zult.Ch {
		dat := granuleToMap(granule, fields)
		row := table.Row{}
		for _, field := range fields {
			val := dat[field]
			if field == "provider_dates" {
				s := ""
				for k, v := range granule.ProviderDates {
					s += fmt.Sprintf("%s: %v\n", k, v)
				}
				val = strings.TrimSpace(s)
			}
			row = append(row, val)
		}
		t.AppendRow(row)
	}
	t.Render()

	return zult.Err()
}

func tablesWriter(zult internal.GranuleResult, w io.Writer, fields []string) error {
	for granule := range zult.Ch {
		t := table.NewWriter()
		t.SetOutputMirror(w)
		t.SetStyle(table.StyleLight)
		dat := granuleToMap(granule, fields)
		for _, field := range fields {
			val := dat[field]
			if field == "provider_dates" {
				s := ""
				for k, v := range granule.ProviderDates {
					s += fmt.Sprintf("%s: %v\n", k, v)
				}
				val = s
			}
			t.AppendRow(table.Row{field, val})
		}
		t.Render()
	}
	return zult.Err()
}

func jsonWriter(zult internal.GranuleResult, w io.Writer, fields []string) error {
	enc := json.NewEncoder(w)
	for granule := range zult.Ch {
		err := enc.Encode(granuleToMap(granule, fields))
		if err != nil {
			panic("encoding output: " + err.Error())
		}
	}
	return zult.Err()
}

func csvWriter(zult internal.GranuleResult, w io.Writer, fields []string) error {
	w.Write([]byte(strings.Join(fields, ",") + "\n"))

	for granule := range zult.Ch {
		vals := []string{}
		m := granuleToMap(granule, fields)
		for _, name := range fields {
			vals = append(vals, fmt.Sprintf("%v", m[name]))
		}
		w.Write([]byte(strings.Join(vals, ",") + "\n"))
	}
	return zult.Err()
}

// FIXME: Uhg! This is so ugly. Need a better way to map granule to fields. Consider mapstructure.
func granuleToMap(gran internal.Granule, fields []string) map[string]any {
	haveField := map[string]bool{}
	for _, name := range fields {
		haveField[name] = true
	}

	dat, err := json.Marshal(gran)
	if err != nil {
		panic("json marshalling error:" + err.Error())
	}

	var mapDat map[string]any
	err = json.Unmarshal(dat, &mapDat)
	if err != nil {
		panic("json unmarshal error: " + err.Error())
	}

	for name := range mapDat {
		if !haveField[name] {
			delete(mapDat, name)
		}
	}
	return mapDat
}
