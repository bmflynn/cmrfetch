package granules

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/bmflynn/cmrfetch/internal"
)

type outputWriter func(internal.GranuleResult, io.Writer, []string) error

func longWriter(zult internal.GranuleResult, w io.Writer, fields []string) error {
	return nil
}

func tablesWriter(zult internal.GranuleResult, w io.Writer, fields []string) error {
	for granule := range zult.Ch {
		t := table.NewWriter()
		t.SetOutputMirror(w)
    t.SetStyle(table.StyleLight)
    dat := granuleToMap(granule, fields) 
    for _, field := range fields {
      t.AppendRow(table.Row{field, dat[field]})
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
			vals = append(vals, m[name])
		}
		w.Write([]byte(strings.Join(vals, ",") + "\n"))
	}
	return zult.Err()
}

func granuleToMap(gran internal.Granule, fields []string) map[string]string {
	haveField := map[string]bool{}
	for _, name := range fields {
		haveField[name] = true
	}

	dat, err := json.Marshal(gran)
	if err != nil {
		panic("json marshalling error:" + err.Error())
	}

	var mapDat map[string]string
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
