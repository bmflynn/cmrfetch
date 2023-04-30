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
  Short: "Search for NASA Earthdata collections and download view associated granules",
	Long: `
Search for NASA Earthdata collections and download view associated granules.

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
  Example: `
  Search for all products with a collection short name prefix:

    cmrfetch collections -s CLDMSK_*

  Search for multiple collection short names:

    cmrfetch collections -s CLDMSK_L2_VIIRS_SNPP -s CLDMSK_L2_VIIRS_NOAA20,CLDMSK_L2_MODIS_Aqua

  Search for a collection by keyword:

    cmrfetch collections -k aerdt

  Search for granules:
  
  `,
	Version:      internal.Version,
  CompletionOptions: cobra.CompletionOptions{
    HiddenDefaultCmd: true,
  },
}

func init() {
	rootCmd.AddCommand(collections.Cmd)
	rootCmd.AddCommand(granules.Cmd)
}

func Execute() error {
	return rootCmd.Execute()
}
