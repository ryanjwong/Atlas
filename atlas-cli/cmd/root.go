package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "1.0.0"

var (
	verbose bool
	output  string
)

var rootCmd = &cobra.Command{
	Use:     "atlas-cli",
	Short:   "Atlas CLI - A command line interface for Atlas",
	Long:    `Atlas CLI is a command line interface that automates your entire software development lifecycle.`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "text", "Output format (text, json)")
}

func GetVerbose() bool {
	return verbose
}

func GetOutput() string {
	return output
}

func GetVersion() string {
	return version
}
