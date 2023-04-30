package collections

import (
	"io"

	"github.com/bmflynn/cmrfetch/internal"
	"github.com/jedib0t/go-pretty/v6/table"
)

type outputWriter func(internal.CollectionResult, io.Writer) error

func tableWriter(zult internal.CollectionResult, w io.Writer) error {
	t := table.NewWriter()
	t.SetOutputMirror(w)
  t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"shortname", "version", "concept", "revision_id", "provider"})

	for col := range zult.Ch {
		t.AppendRow(table.Row{
			col["shortname"],
			col["version"],
			col["concept_id"],
      col["revision_id"],
			col["provider"],
		})
	}

	t.Render()
	return zult.Err()
}

func writeCollection(zult internal.CollectionResult, w io.Writer, long bool) error {
	fields := []string{
		"shortname",
		"version",
		"processing_level",
		"instruments",
		"concept_id",
		"doi",
		"provider",
		"revision_id",
		"revision_date",
		"infourls",
	}

	for col := range zult.Ch {
		t := table.NewWriter()
		t.SetOutputMirror(w)
    t.SetStyle(table.StyleLight)
		t.SetTitle(col["title"])

		for _, name := range fields {
			t.AppendRow(table.Row{name, col[name]})
		}
		if long {
			t.SetCaption(col["abstract"])
		}
		t.Render()
		w.Write([]byte{'\n'})
	}
	return zult.Err()
}

func shortWriter(zult internal.CollectionResult, w io.Writer) error {
	return writeCollection(zult, w, false)
}

func longWriter(zult internal.CollectionResult, w io.Writer) error {
	return writeCollection(zult, w, true)
}
