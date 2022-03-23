package cmd

import (
	"html/template"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"gitlab.ssec.wisc.edu/brucef/cmrfetch/internal"
)

var Collections = &cobra.Command{
	Use:   "collections",
	Short: "List collection metadata",
	Run: func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		provider, err := flags.GetString("provider")
		if err != nil {
			panic(err)
		}
		shortName, err := flags.GetString("shortname")
		if err != nil {
			panic(err)
		}
		if provider == "" && shortName == "" {
			return
		}

		api := internal.NewCMRAPI()
		collections, err := api.Collections(provider, shortName)
		if err != nil {
			log.WithError(err).Fatal("failed to collect")
		}
		t := template.Must(template.New("").Parse(collectionListTmpl))
		if err := t.Execute(os.Stdout, collections); err != nil {
			log.WithError(err).Fatal("template error")
		}
	},
	SilenceUsage: true,
}

func init() {
	flags := Collections.Flags()
	flags.String("provider", "", "Provider name; `data_center` in the collection metadata.")
	flags.String("shortname", "", "ShortName; `short_name` in the collection metadata.")
}

var collectionListTmpl = `
ID                  ShortName                     Version  DatasetID   
============================================================================================================
{{ range . -}}
{{ printf "%-20s" .ID }}{{ printf "%-30s" .ShortName }}{{ printf "%-9s" .Version }}{{ .DatasetID }}
{{ end -}}
===============
Total: {{ len . }}
`
