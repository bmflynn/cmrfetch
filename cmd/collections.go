package cmd

import (
	"fmt"
	"html/template"
	"os"

	"github.com/spf13/cobra"
	"gitlab.ssec.wisc.edu/brucef/cmrfetch/internal"
)

var Collections = &cobra.Command{
	Use:   "collections",
	Short: "List collections",
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
		collections, errs := api.Collections(provider, shortName)
		t := template.Must(template.New("").Parse(collectionListTmpl))
		tmplErr := t.Execute(os.Stdout, collections)

		// return api error first as that's more likely to be an issue
		if apiErr := <-errs; apiErr != nil {
			fmt.Printf("failed handling api response: %s", err)
			os.Exit(1)
		}

		if tmplErr != nil {
			fmt.Printf("failed rendering output: %s", tmplErr)
			os.Exit(1)
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
{{ end }}
`
