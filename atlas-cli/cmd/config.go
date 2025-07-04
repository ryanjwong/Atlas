package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage Atlas CLI configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := map[string]any{
			"verbose": GetVerbose(),
			"output":  GetOutput(),
			"version": GetVersion(),
		}

		if GetOutput() == "json" {
			jsonOutput, _ := json.MarshalIndent(config, "", "  ")
			fmt.Println(string(jsonOutput))
		} else {
			fmt.Printf("Verbose: %t\n", config["verbose"])
			fmt.Printf("Output Format: %s\n", config["output"])
			fmt.Printf("Version: %s\n", config["version"])
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
}