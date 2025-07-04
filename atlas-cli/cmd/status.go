package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current status",
	Long:  `Display the current status of Atlas CLI and related services.`,
	Run: func(cmd *cobra.Command, args []string) {
		if GetVerbose() {
			fmt.Println("Running status command with verbose output...")
		}

		status := map[string]any{
			"status":  "running",
			"message": "All systems operational",
		}

		if len(args) > 0 {
			status["args"] = args
		}

		if GetOutput() == "json" {
			jsonOutput, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(jsonOutput))
		} else {
			fmt.Printf("Status: %s\n", status["status"])
			fmt.Printf("Message: %s\n", status["message"])
			if len(args) > 0 {
				fmt.Printf("Additional arguments: %v\n", args)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}