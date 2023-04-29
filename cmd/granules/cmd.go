package granules

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

var (
	timerange     internal.TimeRangeValue
	defaultFields = []string{"name", "size", "checksum", "download_url"}
	validFields   = []string{
		"name", "size", "checksum", "checksum_alg", "download_url", "native_id", "revision_id",
		"concept_id", "collection", "download_direct_url",
	}
)

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func arrayContains[T comparable](arr []T, val T) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

var Cmd = &cobra.Command{
	Use:     "granules (--collection=COL|--nativeid=ID|--shortname=NAME) [flags]",
	Aliases: []string{"g", "gr", "gran", "granule"},
	Short:   "Search for and download collection granules",
	Long: `
Search for and download collection granules

NASA Earthdata Authentication
  Most, if not all, providers require authentication which is generally provided 
  via a netrc file, something like so:

    machine urs.earthdata.nasa.gov login <username> password <plain text password>

  NOTE: It is very important that this file is only accessible via your user. On
  Linux and OSX this can be done via 'chmod 0600 ~/.netrc'.

  For more details:

    https://wiki.earthdata.nasa.gov/display/EL/How+To+Access+Data+With+cURL+And+Wget

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		if !flags.Changed("collection") &&
			!flags.Changed("nativeid") &&
			!flags.Changed("shortname") {
			return fmt.Errorf("at least one of --collection, --shortname, or --nativeid is required")
		}

		netrc, err := flags.GetBool("netrc")
		failOnError(err)
		verbose, err := flags.GetBool("verbose")
		failOnError(err)
		output, err := flags.GetString("output")
		failOnError(err)

		yes, err := flags.GetBool("yes")
		failOnError(err)
		destdir, err := flags.GetString("download")
		failOnError(err)
		concurrency, err := flags.GetInt("download-concurrency")
		failOnError(err)
		clobber, err := flags.GetBool("download-clobber")
		failOnError(err)

		fields, err := flags.GetStringSlice("fields")
		failOnError(err)
		for _, name := range fields {
			if !arrayContains(validFields, name) {
				return fmt.Errorf("%s is not a valid field name", name)
			}
		}

		params, err := newParams(flags)
		if err != nil {
			return err
		}

		var logger *log.Logger
		if verbose {
			logger = log.New(os.Stderr, "", log.LstdFlags)
		}

		api := internal.NewCMRSearchAPI(logger)

		var writer outputWriter
		switch output {
    case "tables": 
      writer = tablesWriter
		case "json":
			writer = jsonWriter
		case "csv":
			writer = csvWriter
		default:
			return fmt.Errorf("--output must be one of tables, json, csv")
		}

		if destdir != "" {
			err = doDownload(context.TODO(), api, params, destdir, netrc, clobber, yes, verbose, concurrency)
		} else {
			err = do(api, params, writer, fields, yes)
		}
		if err != nil {
			log.Fatalf("failed! %s", err)
		}

		return nil
	},
}

func init() {
	flags := Cmd.Flags()

	flags.BoolP("verbose", "v", false, "Verbose output")
	flags.BoolP("yes", "y", false, "Answer yes to any prompts when using --download.")
	flags.String("download", "",
		"Download the resulting granules to the directory provided. If the directory does not "+
			fmt.Sprintf("exist it will be created. More than %v total granules in the ", maxResultsWithoutPrompt)+
			"result set will require confirmation, which can be skipped using --yes. By default, "+
			"If a file exists in the destination directory it will be skipped; see --download-clobber. "+
			"Checksums are verified for all downloaded files, if a checksum is available.")
	flags.BoolP("download-clobber", "C", false, "Overwrite any existing files when downloading.")
	flags.Int("download-concurrency", defaultDownloadConcurrency, "Number of concurrent downloads")
	flags.Bool("netrc", true,
		"Use netrc for basic authentication credentials on redirect. This is necessary for NASA "+
			"Earthdata authentication, which many providers use. See the NASA Earthdata Authentication "+
			"above.")

	flags.StringSliceP("nativeid", "N", nil, "granule native id")
	flags.StringSliceP("collection", "c", nil, "Collection concept id")
	flags.StringSliceP("shortname", "n", nil, "Collection short name")
	flags.VarP(&timerange, "timerange", "t", "Timerange as <start>,[<end>]")
	// FIXME: Have to sort by revision_date to make sure we don'get get errors regaring concept-id and revision
	//        This seems to be a CMR issue, but I'm not really sure.
	// flags.StringP("sortby", "S", "",
	// 	fmt.Sprintf("Sort by one of %s. Prefix the field name by `-` to sort descending", strings.Join(sortByFields, ", ")))
	flags.Float64Slice("polygon", nil,
		"Polygon points are provided in counter-clockwise order. The last point should match the first point to "+
			"close the polygon. The values are listed comma separated in longitude latitude order, "+
			"i.e. lon1, lat1, lon2, lat2, lon3, lat3, and so on.")
	flags.Float64Slice("bouding-box", nil, "Granules overlapping a bounding box, where the corner "+
		"points are provided lon1,lat1,lon2,lat2.")
	flags.Float64Slice("circle", nil, "Granules overlapping a circle, where the circle is defined as "+
		"centerlon,centerlat,radius.")
	flags.Float64Slice("point", nil, "Granules containing point lon,lat.")
	flags.StringSlice("fields", defaultFields,
		"Fields to include in output. "+strings.Join(validFields, ", "))

	flags.StringP("output", "o", "tables", "Output format. One of tables, json, or, csv")
}

func do(api *internal.CMRSearchAPI, params internal.SearchGranuleParams, writer outputWriter, fields []string, yes bool) error {
	zult, err := api.SearchGranules(context.Background(), params)
	if err != nil {
		return err
	}

	return writer(zult, os.Stdout, fields)
}

func newParams(flags *pflag.FlagSet) (internal.SearchGranuleParams, error) {
	params := internal.SearchGranuleParams{}

	s, err := flags.GetStringSlice("collection")
	failOnError(err)
	params.Collection(s...)

	s, err = flags.GetStringSlice("nativeid")
	failOnError(err)
	params.NativeID(s...)

	s, err = flags.GetStringSlice("shortname")
	failOnError(err)
	params.ShortName(s...)

	if flags.Changed("timerange") {
		failOnError(err)
		params.Timerange(*timerange.Start, timerange.End)
	}

	a, err := flags.GetFloat64Slice("polygon")
	failOnError(err)
	params.Polygon(a)

	a, err = flags.GetFloat64Slice("bouding-box")
	failOnError(err)
	params.BoundingBox(a)

	a, err = flags.GetFloat64Slice("circle")
	failOnError(err)
	params.Circle(a)

	a, err = flags.GetFloat64Slice("point")
	failOnError(err)
	params.Point(a)

	return params, nil
}
