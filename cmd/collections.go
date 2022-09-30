package cmd

import (
	"encoding/json"
	"html/template"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"gitlab.ssec.wisc.edu/brucef/cmrfetch/internal"
)

type simpleCollection struct {
	ConceptID  string `json:"concept_id"`
	ShortName  string `json:"short_name"`
	RevisionID int    `json:"revision_id"`
	Version    string `json:"version"`
	Title      string `json:"title"`
}

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
		output, err := flags.GetString("output")
		if err != nil {
			panic(err)
		}
		if output != "json" && output != "simple" {
			log.Fatalf("invalid output type")
		}

		api := internal.NewCMRAPI()
		collections, err := api.Collections(provider, shortName)
		if err != nil {
			log.WithError(err).Fatal("failed to collect")
		}

		if output == "json" {
			simpleCollections := []simpleCollection{}
			for _, col := range collections {
				simpleCollections = append(simpleCollections, simpleCollection{
					ConceptID:  col.Meta.ConceptID,
					ShortName:  col.ShortName,
					RevisionID: col.Meta.RevisionID,
					Version:    col.Version,
					Title:      col.EntryTitle,
				})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(simpleCollections); err != nil {
				log.WithError(err).Fatal("template error")
			}
		} else {
			t := template.Must(template.New("").Parse(collectionListTmpl))
			if err := t.Execute(os.Stdout, collections); err != nil {
				log.WithError(err).Fatal("template error")
			}
		}

	},
	SilenceUsage: true,
}

func init() {
	flags := Collections.Flags()
	flags.String("provider", "", "Provider name, e.g., ASIPS, LAADS, etc...")
	flags.String("shortname", "", "ShortName; `short_name` in the collection metadata.")
	flags.StringP("output", "o", "simple", "Output type. Valid values include json, simple")
}

var collectionListTmpl = `
ID                   ShortName                      Version  Revision  Description
============================================================================================================
{{ range . -}}
{{ printf "%-20s" .Meta.ConceptID }} {{ printf "%-30s" .ShortName }} {{ printf "%-8v" .Version }} {{ printf "%-9v" .Meta.RevisionID }} {{ .EntryTitle }}
{{ end -}}
===============
Total: {{ len . }}
`
