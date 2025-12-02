package main

import (
	"os"

	"github.com/mm/background-coding-agent/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
