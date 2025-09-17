package main

import (
	"os"

	"github.com/ccheshirecat/viper/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
