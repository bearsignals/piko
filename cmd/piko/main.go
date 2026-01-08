package main

import (
	"os"

	"github.com/gwuah/piko/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
