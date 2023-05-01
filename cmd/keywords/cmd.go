package keyworkds

import (
	"context"
	"log"
	"os"

	"github.com/bmflynn/cmrfetch/internal"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	flags := Cmd.Flags()
	flags.BoolP("verbose", "v", false, "Verbose output")
}

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func elipsis(s string, maxLen int) string {
	if len(s) < maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

var Cmd = &cobra.Command{
	Use:     "keywords <query>",
	Aliases: []string{"k", "kw", "key", "keyword"},
	Args:    cobra.ExactArgs(1),
	Short:   "Search for permissible names for platform, instrument, and provider",
	Long: `

Using this search can be hit-or-miss, as not all platforms, instruments, and 
providers are registered. However, it can be useful for those that are registered 
if you know a few keywords regarding what you're looking for. 

For example, if you need to know the name of the Suomi-NPP platform (sometimes 
referred to as NPP) you can perform a search using the query "npp" to get the
correct value.
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, err := cmd.Flags().GetBool("verbose")
		failOnError(err)

		var logger *log.Logger
		if verbose {
			logger = log.New(os.Stderr, "", log.LstdFlags)
		}

		api := internal.NewCMRSearchAPI(logger)

		zult, err := api.SearchFacets(context.Background(), args[0], nil)
		if err != nil {
			log.Fatalf("Failed! %s", err)
		}

		t := table.NewWriter()
		t.SetStyle(table.StyleLight)
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Type", "Value", "Score", "Fields"})

		for facet := range zult.Ch {
			t.AppendRow(table.Row{facet.Type, elipsis(facet.Value, 48), facet.Score, elipsis(facet.Fields, 48)})
		}
		t.Render()

		if err := zult.Err(); err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		return nil
	},
}
