package main

import (
	"github.com/spf13/cobra"
	"gitlab.ssec.wisc.edu/brucef/cmrfetch/cmd"
)

var version = "<notset>"

var root = &cobra.Command{
	Use:   "cmrfetch",
	Short: "Ingest data from NASA CMR",
	Long: `Ingest data from NASA CMR.
	
See the list of sub-commands below for more information.

There are a few assuptions with regards to collection metadata made by this tool that 
may not necessarilly be true for all CMR collections. If it does not work for a 
collection that you are interested in please create an issue using one of the links
below.

Project: https://github.com/bmflynn/cmrfetch
`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version: version,
}

func init() {
	root.AddCommand(cmd.Collections)
	root.AddCommand(cmd.Granules)
	root.AddCommand(cmd.Ingest)
}

func main() {
	root.Execute()
}
