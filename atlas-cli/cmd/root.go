package cmd

import (
	"fmt"
	"os"

	"github.com/ryanjwong/Atlas/atlas-cli/internal/services"
	"github.com/spf13/cobra"
)

const version = "1.0.0"

var (
	verbose bool
	output  string
	svc     *services.Services
)

var rootCmd = &cobra.Command{
	Use:     "atlas-cli",
	Short:   "Atlas CLI - A command line interface for Atlas",
	Long:    `Atlas CLI is a command line interface that automates your entire software development lifecycle.`,
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		svc, err = services.NewServices(verbose, output, version, "./state.db")
		if err != nil {
			panic(err)
		}
		return nil
	},
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

func GetServices() *services.Services {
	return svc
}

func GetVerbose() bool {
	if svc != nil {
		return svc.GetVerbose()
	}
	return verbose
}

func GetOutput() string {
	if svc != nil {
		return svc.GetOutput()
	}
	return output
}

func GetVersion() string {
	return version
}
