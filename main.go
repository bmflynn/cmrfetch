package main

import (
	"gitlab.ssec.wisc.edu/brucef/cmrfetch/cmd"
)

var (
	version = "<notset>"
	root    = cmd.Ingest
)

func init() {
	root.Version = version
	root.AddCommand(cmd.Collections)
	root.AddCommand(cmd.Granules)
}

func main() {
	root.Execute()
}
