package main

import (
	"fmt"

	"github.com/bmflynn/cmrfetch/cmd"
)


func main() {
  if err := cmd.Execute(); err != nil {
    fmt.Println("Use --help for more information")
  }
}
