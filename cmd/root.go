package cmd

import (
	"github.com/bmflynn/cmrfetch/cmd/collections"
	"github.com/bmflynn/cmrfetch/cmd/granules"
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
	Short: "Search for NASA Earthdata CMR products and collections.",
	Long: `
View NASA CMR Collection and granule metdata and download products.

Use the collections subcommand to search for and discover NASA Earthdata collections
available via the CMR Search API.

References:

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
	Version:      internal.Version,
}

func init() {
	rootCmd.AddCommand(collections.Cmd)
	rootCmd.AddCommand(granules.Cmd)
}

func Execute() error {
	return rootCmd.Execute()
}
