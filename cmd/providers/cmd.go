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
	Short:   "List EOSDIS providers",
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
	providers, err := internal.GetProviderHoldings()
	if err != nil {
		return fmt.Errorf("fetching provider holdings: %w", err)
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"name", "collections", "granules"})
	for _, provider := range providers {
		t.AppendRow(table.Row{provider.ID, len(provider.Collections), provider.GranuleCount})
	}
	t.Render()
	return nil
}
