package providers

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/bmflynn/cmrfetch/internal"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

var Cmd = &cobra.Command{
	Use:     "providers",
	Aliases: []string{"p", "pr", "prov", "provider"},
	Short:   "List EOSDIS providers and general collection metadata",
  Long: `List metadata on providers and their collections. 

This is mostly useful for to discover provider names. Once you have a provider name it 
is generally easier and faster to use the collections command.

NOTE: This downloads a large JSON database from CMR and the initial download can 
sometimes take a long time. The result is cached for 30d after initial download.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := cmd.Flags().GetStringSlice("name")
		failOnError(err)

		if err := do(names); err != nil {
			log.Fatalf("failed! %s", err)
		}

		return nil
	},
}

func init() {
	flags := Cmd.Flags()

	flags.StringSliceP("name", "n", nil, "List collections available for providers with given name(s)")
}

func do(names []string) error {
	allProviders, err := internal.GetProviderHoldings()
	if err != nil {
		return fmt.Errorf("fetching provider holdings: %w", err)
	}

	sort.Slice(allProviders, func(i, j int) bool {
		return allProviders[i].ID < allProviders[j].ID
	})

  if len(names) > 0 {
    return doCollections(names, allProviders)
  } else {
    return doProviders(allProviders)
  }
}

func doProviders(allProviders []internal.Provider) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"name", "collections", "granules"})
	for _, provider := range allProviders {
		t.AppendRow(table.Row{provider.ID, len(provider.Collections), provider.GranuleCount})
	}
	t.Render()
	return nil
}

func doCollections(names []string, allProviders []internal.Provider) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"concept-id", "granules", "title"})
  if len(names) > 0 {
    wantNames := map[string]bool{}
    for _, n := range names {
      wantNames[n] = true
    }
    for _, p := range allProviders {
      if wantNames[p.ID] {
        for _, c := range p.Collections {
          t.AppendRow(table.Row{c.ConceptID, c.GranuleCount, c.Title})
        }
      }
    }
  }
	t.Render()
  return nil
}
