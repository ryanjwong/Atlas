package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/providers"
	"github.com/spf13/cobra"
)

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage clusters",
	Long:  `Create, delete, and manage Kubernetes clusters across cloud providers.`,
}

var clusterCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new cluster",
	Long:  `Create a new Kubernetes cluster with the specified name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		clusterName := args[0]
		services.Log(fmt.Sprintf("Creating cluster: %s", clusterName))

		provider, _ := cmd.Flags().GetString("provider")
		region, _ := cmd.Flags().GetString("region")
		nodeCount, _ := cmd.Flags().GetInt("nodes")
		var p providers.Provider
		switch provider {
		case "local":
			localProvider := services.GetLocalProvider()
			p = &localProvider
		default:
			return fmt.Errorf("unsupported provider: %s", provider)
		}
		
		_, err := p.CreateCluster(context.Background(), &providers.ClusterConfig{
			Name:      clusterName,
			Region:    region,
			NodeCount: nodeCount,
		})
		if err != nil {
			return fmt.Errorf("failed to create cluster: %w", err)
		}
		services.Log("Cluster creation initiated successfully")
		return nil
	},
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clusters",
	Long:  `List all clusters managed by Atlas CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		services.Log("Listing clusters")
		clusters, err := services.GetStateManager().ListClusters(context.Background())

		if err != nil {
			return fmt.Errorf("error listing clusters: %s", err)
		}

		if len(clusters) == 0 {
			fmt.Println("No clusters found")
			return nil
		}

		if services.GetOutput() == "json" {
			jsonOutput, err := json.MarshalIndent(clusters, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal clusters: %w", err)
			}
			fmt.Println(string(jsonOutput))
		} else {
			fmt.Printf("%-20s %-10s %-15s %-6s %-10s\n", "NAME", "PROVIDER", "REGION", "NODES", "STATUS")
			fmt.Printf("%-20s %-10s %-15s %-6s %-10s\n", "----", "--------", "------", "-----", "------")
			for _, cluster := range clusters {
				fmt.Printf("%-20s %-10s %-15s %-6v %-10s\n",
					cluster.Name,
					cluster.Provider,
					cluster.Region,
					cluster.NodeCount,
					cluster.Status)
			}
		}

		services.Log("Listed clusters successfully")
		return nil
	},
}

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a cluster",
	Long:  `Delete a Kubernetes cluster by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		clusterName := args[0]
		services.Log(fmt.Sprintf("Deleting cluster: %s", clusterName))

		localProvider := services.GetLocalProvider()
		p := &localProvider
		err := p.DeleteCluster(context.Background(), clusterName)
		if err != nil {
			return fmt.Errorf("failed to delete cluster: %w", err)
		}

		result := map[string]any{
			"name":    clusterName,
			"status":  "deleted",
			"message": fmt.Sprintf("Cluster '%s' deleted successfully", clusterName),
		}

		if services.GetOutput() == "json" {
			jsonOutput, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			fmt.Println(string(jsonOutput))
		} else {
			fmt.Printf("Cluster '%s' deleted successfully\n", clusterName)
		}

		services.Log("Cluster deletion completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)

	clusterCreateCmd.Flags().StringP("provider", "p", "local", "Cloud provider (local, aws, gcp, azure)")
	clusterCreateCmd.Flags().StringP("region", "r", "us-west-2", "Region to create cluster in")
	clusterCreateCmd.Flags().IntP("nodes", "n", 3, "Number of nodes in the cluster")
}
