package granules

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bmflynn/cmrfetch/internal"
	"github.com/bmflynn/cmrfetch/internal/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	timerange   internal.TimeRangeValue
	validFields = []string{
		"name", "size", "checksum", "checksum_alg", "download_url", "native_id", "revision_id",
		"concept_id", "collection", "download_direct_url", "daynight", "timerange", "boundingbox",
		"provider_dates",
	}
	defaultFields = validFields
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
	Example: `
  Search for all products with a collection short name prefix:

    cmrfetch granules -s "CLDMSK_*"

  Search for multiple collection short names:

    cmrfetch granules -s CLDMSK_L2_VIIRS_SNPP -s CLDMSK_L2_VIIRS_NOAA20,CLDMSK_L2_MODIS_Aqua

  Search for granules by filename:

    cmrfetch granules -c C1964798938-LAADS -f CLDMSK_L2_VIIRS_NOAA20.A2023115.0142.001.2023115140055.nc 
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()

		if !flags.Changed("collection") &&
			!flags.Changed("nativeid") &&
			!flags.Changed("shortname") &&
			!flags.Changed("filename") {
			return fmt.Errorf("at least one of --collection, --shortname, --nativeid, or --filename is required")
		}

		token, err := flags.GetString("edltoken")
		failOnError(err)
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
		downloadSkipChecksum, err := flags.GetBool("download-skip-checksum")
		failOnError(err)

		fields, err := flags.GetStringSlice("fields")
		failOnError(err)
		for _, name := range fields {
			if !arrayContains(validFields, name) {
				return fmt.Errorf("%s is not a valid field name", name)
			}
		}

		if flags.Changed("filename") && !flags.Changed("collection") {
			return fmt.Errorf("--collection is required when using --filename")
		}

		params, err := newParams(flags)
		if err != nil {
			return err
		}

		log.SetVerbose(verbose)

		api := internal.NewCMRSearchAPI()

		if destdir != "" {
			err = doDownload(context.TODO(), api, params, destdir, token, netrc, clobber, yes, downloadSkipChecksum, concurrency)
		} else {
			err = do(api, params, output, fields)
		}
		if err != nil {
			log.Fatalf("failed! %s", err)
		}

		return nil
	},
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.BoolP("verbose", "v", false, "Verbose output")
	flags.BoolP("yes", "y", false, "Answer yes to any prompts when using --download.")

	flags.StringP("download", "d", "",
		"Download the resulting granules to the directory provided. If the directory does not "+
			fmt.Sprintf("exist it will be created. More than %v total granules in the ", maxResultsWithoutPrompt)+
			"result set will require confirmation, which can be skipped using --yes. By default, "+
			"If a file exists by name in the destination directory it will be skipped; see --download-clobber. "+
			"Checksums are verified for all downloaded files, if a checksum is available.")
	flags.BoolP("download-clobber", "C", false, "Overwrite any existing files when downloading.")
	flags.Int("download-concurrency", defaultDownloadConcurrency, "Number of concurrent downloads")
	flags.BoolP(
		"download-skip-checksum",
		"S",
		false,
		"Enable skipping files where the remote checksum does not match the local checksum in addition "+
			"to skipping files that exist by name. If the there is no checksum available in the remote "+
			"metadata, exists checking is done by name only.",
	)
	flags.String("edltoken", "",
		"Use a NASA EDL token for bearer-based authentication on redirect. Either this or netrc is "+
			"necessary for NASA Earthdata authentication, which many providers use. See the NASA "+
			"Earthdata Authentication above.")
	flags.Bool("netrc", true,
		"Use netrc for basic authentication credentials on redirect. Either this or edltoken is "+
			"necessary for NASA Earthdata authentication, which many providers use. See the NASA "+
			"Earthdata Authentication above.")

	flags.StringSliceP("nativeid", "N", nil, "granule native id")
	flags.StringSliceP("collection", "c", nil, "Collection concept id")
	flags.StringSliceP("shortname", "s", nil, "Collection short name")
	flags.StringSliceP("filename", "f", nil,
		"Filter on an approximation of the filename. Must be sepcified with --collection. In CMR metadata "+
			"terms this searches the granule ur and producer granule id.")
	flags.StringP("daynight", "D", "", "Day or night grnaules. One of day, night, both, or unspecified")
	flags.VarP(&timerange, "timerange", "t", "Timerange as <start>,[<end>]")
	flags.Float64Slice("polygon", nil,
		"Polygon points are provided in counter-clockwise order. The last point should match the first point to "+
			"close the polygon. The values are listed comma separated in longitude latitude order, "+
			"i.e. lon1,lat1,lon2,lat2,lon3,lat3, and so on.")
	flags.Float64Slice("bounding-box", nil, "Granules overlapping a bounding box, where the corner "+
		"points are provided lon1,lat1,lon2,lat2.")
	flags.Float64Slice("circle", nil, "Granules overlapping a circle, where the circle is defined as "+
		"centerlon,centerlat,radius.")
	flags.Float64Slice("point", nil, "Granules containing point lon,lat.")
	flags.StringSlice("fields", defaultFields,
		"Fields to include in output; ignored for --output=short. "+strings.Join(validFields, ", "))
	flags.StringP("output", "o", "short",
		"Output format. One of short, long, json, or, csv. The default output does not handle paged "+
			"results and must load all results in memory before rendering. Make sure to provide enough "+
			"filters to limit the result set to a reasonable size or use json or csv output.")

	cobra.CheckErr(flags.MarkHidden("shortname"))
	cobra.CheckErr(flags.MarkDeprecated("yes", "Not used and will be ignored"))
	cobra.CheckErr(flags.MarkDeprecated("shortname", "Provide the collection concept id instead"))
}

func do(api *internal.CMRSearchAPI, params *internal.SearchGranuleParams, writerName string, fields []string) error {
	var writer outputWriter
	switch writerName {
	case "short":
		writer = shortWriter
	case "long":
		writer = tablesWriter
	case "json":
		writer = jsonWriter
	case "csv":
		writer = csvWriter
	default:
		return fmt.Errorf("--output must be one of short, long, json, csv")
	}

	zult, err := api.SearchGranules(context.Background(), params)
	if err != nil {
		return err
	}

	if writerName == "short" && zult.Hits() > 1000 {
		log.Printf(
			"WARNING: short output renders in memory and you have more than 1000 results. " +
				"Consider limiting your search to reduce the number of results or use json or csv " +
				"output.")
	}

	return writer(zult, os.Stdout, fields)
}

func newParams(flags *pflag.FlagSet) (*internal.SearchGranuleParams, error) {
	params := &internal.SearchGranuleParams{}

	if flags.Changed("daynight") {
		st, err := flags.GetString("daynight")
		failOnError(err)
		if ok, _ := regexp.MatchString(`^(day|night|both|unspecified)$`, st); !ok {
			return params, fmt.Errorf("daynight must be one of day, night, both, or unspecified")
		}
		params.DayNightFlag(st)
	}

	if flags.Changed("collection") {
		sa, err := flags.GetStringSlice("collection")
		failOnError(err)
		params.Collections(sa...)
	}

	if flags.Changed("nativeid") {
		sa, err := flags.GetStringSlice("nativeid")
		failOnError(err)
		params.NativeIDs(sa...)
	}

	if flags.Changed("shortname") {
		sa, err := flags.GetStringSlice("shortname")
		failOnError(err)
		params.ShortNames(sa...)
	}

	if flags.Changed("filename") {
		sa, err := flags.GetStringSlice("filename")
		failOnError(err)
		params.Filenames(sa...)
	}

	if flags.Changed("timerange") {
		params.Timerange(*timerange.Start, timerange.End)
	} else {
		end := time.Now().UTC()
		start := end.Add(-time.Hour * 24)
		params.Timerange(start, &end)
	}

	a, err := flags.GetFloat64Slice("polygon")
	failOnError(err)
	params.Polygon(a)

	a, err = flags.GetFloat64Slice("bounding-box")
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
