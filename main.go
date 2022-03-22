package main

import (
	"github.com/spf13/cobra"
	"gitlab.ssec.wisc.edu/brucef/cmrfetch/cmd"
)

var version = "<notset>"

var root = &cobra.Command{
	Use:   "cmrsearch",
	Short: "Tools for ingesting data from NASA CMR",
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
