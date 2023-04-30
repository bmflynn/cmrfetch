package collections

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bmflynn/cmrfetch/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

var Cmd = &cobra.Command{
	Use:     "collections",
	Aliases: []string{"c", "col", "collection"},
	Short:   "Search for and discover Collections",
  Example: `
  Search for all products with a collection short name prefix:

    cmrfetch collections -s CLDMSK_*

  Search for multiple collection short names:

    cmrfetch collections -s CLDMSK_L2_VIIRS_SNPP -s CLDMSK_L2_VIIRS_NOAA20,CLDMSK_L2_MODIS_Aqua

  Search for a collection by keyword:

    cmrfetch collections -k aerdt
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()
		output, err := flags.GetString("output")
		failOnError(err)

		params, err := newParams(flags)
		if err != nil {
			return err
		}

		verbose, err := flags.GetBool("verbose")
		failOnError(err)

		var logger *log.Logger
		if verbose {
			logger = log.New(os.Stderr, "", log.LstdFlags)
		}
		api := internal.NewCMRSearchAPI(logger)

		var writer outputWriter
		switch output {
		case "brief":
			writer = tableWriter
		case "short":
			writer = shortWriter
		case "long":
			writer = longWriter
		default:
			return fmt.Errorf("--output must be one of brief, short or long")
		}

		return do(api, params, writer)
	},
}

func init() {
	flags := Cmd.Flags()

	flags.BoolP("verbose", "v", false, "Verbose output")
	flags.StringP("keyword", "k", "",
		"Keyword search or search pattern (supporting ? or *) to search over collection metadata")
	flags.StringSliceP("provider", "P", []string{},
		"Filter on provider name. May be provided more than once or comma separated. "+
			"Example providers include ASIPS or LAADS. For a listing of available providers "+
			"see https://cmr.earthdata.nasa.gov/search/site/collections/directory")
	flags.String("since", "", "Filter to collections that have a revision date greater"+
		"or equal to this UTC time, formatted as <yyyy>-<mm>-<dd>T<hh>:<mm>:<dd>Z")
  flags.StringSliceP("shortname", "s", nil, "Filter on collection short name or pattern (support ? or *)")
	flags.StringSliceP("instrument", "i", []string{},
		"Filter on instrument short name. May be provided more than once or comma separated. "+
			"Common instruments include VIIRS, MODIS, CrIS")
	flags.StringSliceP("platform", "p", []string{},
		"Filter on platform short name. May be provided more than once or comma separated. "+
			"Common platforms: NOAA-21, NOAA-20, Suomi-NPP, Aqua, Terra.")
	flags.StringP("title", "t", "", "Collection title search or search pattern (supporting ? or *)")
	flags.StringP("output", "o", "brief", "Output format. One of table, short, long")
	flags.StringP("sortby", "S", "",
		fmt.Sprintf("Sort by one of %s. Prefix the field name by `-` to sort descending", strings.Join(sortFields, ", ")))
	flags.Bool("cloud-hosted", false,
		"Filter by whether the collection's data is hosted in Earthdata Cloud.")
	flags.Bool("standard", false,
		"Filter to collections tagged as a standard proudct.")
	flags.Bool("has-granules", true,
		"Filter to collections with granules.")
}

func newParams(flags *pflag.FlagSet) (internal.SearchCollectionParams, error) {
	params := internal.NewSearchCollectionParams()

	s, err := flags.GetString("keyword")
	failOnError(err)
	params.Keyword(s)

	s, err = flags.GetString("title")
	failOnError(err)
	params.Title(s)

	s, err = flags.GetString("sortby")
	failOnError(err)
	if s != "" && !validSortField(s) {
		return params, fmt.Errorf("invalid sort field; expected one of %s", strings.Join(sortFields, ", "))
	}
	params.SortBy(s)

	a, err := flags.GetStringSlice("provider")
	failOnError(err)
	params.Providers(a...)

	a, err = flags.GetStringSlice("shortname")
	failOnError(err)
	params.ShortNames(a...)

	a, err = flags.GetStringSlice("platform")
	failOnError(err)
	params.Platforms(a...)

	a, err = flags.GetStringSlice("instrument")
	failOnError(err)
	params.Instruments(a...)

	if flags.Changed("cloud-hosted") {
		b, err := flags.GetBool("cloud-hosted")
		failOnError(err)
		params.CloudHosted(b)
	}

	if flags.Changed("has-granules") {
    b, err := flags.GetBool("has-granules")
		failOnError(err)
		params.HasGranules(b)
	}

	if flags.Changed("standard") {
    b, err := flags.GetBool("standard")
		failOnError(err)
		params.Standard(b)
	}

	return params, nil
}

func do(api *internal.CMRSearchAPI, params internal.SearchCollectionParams, writer outputWriter) error {
	zult, err := api.SearchCollections(context.Background(), params)
	if err != nil {
		return err
	}

	return writer(zult, os.Stdout)
}
