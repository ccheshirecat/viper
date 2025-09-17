package main

import (
	"os"

	"github.com/viper-org/viper/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}