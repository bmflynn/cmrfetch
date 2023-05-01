package cmd

import (
	"github.com/bmflynn/cmrfetch/cmd/collections"
	"github.com/bmflynn/cmrfetch/cmd/granules"
	keyworkds "github.com/bmflynn/cmrfetch/cmd/keywords"
	"github.com/bmflynn/cmrfetch/internal"
	"github.com/spf13/cobra"
)

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "cmrfetch",
	Short: "Search for and download NASA Earthdata collections and granules",
	Long: `
Search for and download NASA Earthdata collections and granules.

References:

  * NASA Eathdata
    https://earthdata.nasa.gov
  * NASA Earthdata CMR Search API:
    https://cmr.earthdata.nasa.gov/search
  * NASA Earthdata Collection Directory:
    https://cmr.earthdata.nasa.gov/search/site/collections/directory/eosdis
    This is particularly useful as a reasonable browseable list of Providers and
    Collections.
  * NASA Global Change Master Directory -- Keywords
    Instruments: https://gcmd.earthdata.nasa.gov/KeywordViewer/scheme/instruments
    Platforms: https://gcmd.earthdata.nasa.gov/KeywordViewer/scheme/platforms

Project: https://github.com/bmflynn/cmrfetch
`,
	Version: internal.Version,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(collections.Cmd)
	rootCmd.AddCommand(granules.Cmd)
	rootCmd.AddCommand(keyworkds.Cmd)
}

func Execute() error {
	return rootCmd.Execute()
}
