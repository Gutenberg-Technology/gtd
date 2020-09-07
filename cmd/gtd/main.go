package main

import (
	"os"
	"strings"

	gtdCobra "github.com/gpkfr/goretdep/cobra"
)

var (
	Version string
)

func main() {
	if strings.EqualFold("", Version) {
		Version = "dev"
	}

	rootCmd := gtdCobra.NewCommand(Version)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
