package main

import (
	"os"

	"github.com/manno/baca/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
