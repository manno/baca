package main

import (
	"os"

	"github.com/manno/background-coding-agent/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
