package main

import (
	"fmt"
	"os"

	"github.com/sunquan/rick/internal/cmd"
)

const VERSION = "2.1.1"

func main() {
	rootCmd := cmd.NewRootCmd(VERSION)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
