package cmd

import (
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"gitlab.ssec.wisc.edu/brucef/cmrfetch/internal"
)

var (
	granuleTimerange = TimerangeVal{}
	sinceTime        *TimeVal
)

var Granules = &cobra.Command{
	Use:   "granules {-c ID|-p PRODUCT}",
	Short: "List granlue metadata",
	Run: func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		id, err := flags.GetString("concept-id")
		if err != nil {
			panic(err)
		}
		product, err := flags.GetString("product")
		if err != nil {
			panic(err)
		}
		productParts := strings.Split(product, "/")
		if product != "" && len(productParts) != 3 {
			fmt.Println("invalid product")
			os.Exit(1)
		}
		if id != "" && product != "" {
			fmt.Println("Cannot specify both --concept-id and --product")
			os.Exit(1)
		}
		header, err := flags.GetBool("header")
		if err != nil {
			panic(err)
		}

		if err := do(id, productParts, (*time.Time)(sinceTime), header); err != nil {
			fmt.Printf("failed! %s\n", err)
			os.Exit(1)
		}
	},
	SilenceUsage: true,
}

func init() {
	flags := Granules.Flags()
	flags.Bool("header", true, "Set to false to hide the result header")
	flags.StringP("concept-id", "c", "", "Concept ID of the collection the granule belongs to.")
	flags.StringP("product", "p", "",
		"Forward slash separated provider, shortname, and version that will be used to lookup the concept id at runtime.")

	flags.VarP(
		sinceTime,
		"since", "s",
		"only granules updated since this tims as  <yyyy-mm-dd>T<hh:mm:ss>Z. "+
			"See https://cmr.earthdata.nasa.gov/search/site/docs/search/api.html#g-updated-since",
	)
	flags.VarP(
		&granuleTimerange, "temporal", "t",
		"Comma separated granule start and end time to search over where time "+
			"format is <yyyy-mm-dd>T<hh:mm:ss>Z. "+
			"See https://cmr.earthdata.nasa.gov/search/site/docs/search/api.html#g-temporal.")

}

var granuleListTmpl = `{{ if .Header -}}
Name                                                        Updated                 URL
=======================================================================================-->
{{- end }}
{{ range .Data -}}
{{ printf "%-60s" .ProducerGranuleID }}{{ .Updated.Format "2006-01-02T15:04:05Z" | printf "%-24s" }}{{ .DownloadURL }}
{{ end -}}
==============
Total: {{ len .Data }}
`

func do(id string, productParts []string, since *time.Time, header bool) error {
	api := internal.NewCMRAPI()

	// Determine the concept id from the parts if provided
	if len(productParts) == 3 {
		col, err := api.Collection(productParts[0], productParts[1], productParts[2])
		if err != nil {
			return fmt.Errorf("collection lookup failed: %w", err)
		}
		id = col.ID
	}

	granules, err := api.Granules(id, granuleTimerange, since)
	if err != nil {
		log.WithError(err).Fatal("failed to fetch granules")
	}
	t := template.Must(template.New("").Parse(granuleListTmpl))
	if err := t.Execute(os.Stdout, struct {
		Data   []internal.Granule
		Header bool
	}{granules, header}); err != nil {
		log.WithError(err).Fatalf("failed to render")
	}

	return nil
}
