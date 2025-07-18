package main

import (
	"os"

	"github.com/conneroisu/templar/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
