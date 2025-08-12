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
		provider, _ := cmd.Flags().GetString("provider")
		var p providers.Provider
		switch provider {
		case "local":
			localProvider := services.GetLocalProvider()
			p = &localProvider
		default:
			return fmt.Errorf("unsupported provider: %s", provider)
		}
		clusters, err := p.ListClusters(context.Background())

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

var clusterStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a cluster",
	Long:  `Start a stopped Kubernetes cluster by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		clusterName := args[0]
		services.Log(fmt.Sprintf("Starting cluster: %s", clusterName))

		localProvider := services.GetLocalProvider()
		p := &localProvider
		err := p.StartCluster(context.Background(), clusterName)
		if err != nil {
			return fmt.Errorf("failed to start cluster: %w", err)
		}

		result := map[string]any{
			"name":    clusterName,
			"status":  "started",
			"message": fmt.Sprintf("Cluster '%s' started successfully", clusterName),
		}

		if services.GetOutput() == "json" {
			jsonOutput, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			fmt.Println(string(jsonOutput))
		} else {
			fmt.Printf("Cluster '%s' started successfully\n", clusterName)
		}

		services.Log("Cluster start completed successfully")
		return nil
	},
}

var clusterStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a cluster",
	Long:  `Stop a running Kubernetes cluster by name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		clusterName := args[0]
		services.Log(fmt.Sprintf("Stopping cluster: %s", clusterName))

		localProvider := services.GetLocalProvider()
		p := &localProvider
		err := p.StopCluster(context.Background(), clusterName)
		if err != nil {
			return fmt.Errorf("failed to stop cluster: %w", err)
		}

		result := map[string]any{
			"name":    clusterName,
			"status":  "stopped",
			"message": fmt.Sprintf("Cluster '%s' stopped successfully", clusterName),
		}

		if services.GetOutput() == "json" {
			jsonOutput, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			fmt.Println(string(jsonOutput))
		} else {
			fmt.Printf("Cluster '%s' stopped successfully\n", clusterName)
		}

		services.Log("Cluster stop completed successfully")
		return nil
	},
}

var clusterScaleCmd = &cobra.Command{
	Use:   "scale [name]",
	Short: "Scale a cluster",
	Long:  `Scale a Kubernetes cluster by changing the number of nodes.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		clusterName := args[0]
		nodeCount, _ := cmd.Flags().GetInt("nodes")
		
		services.Log(fmt.Sprintf("Scaling cluster: %s to %d nodes", clusterName, nodeCount))

		localProvider := services.GetLocalProvider()
		p := &localProvider
		err := p.ScaleCluster(context.Background(), clusterName, nodeCount)
		if err != nil {
			return fmt.Errorf("failed to scale cluster: %w", err)
		}

		result := map[string]any{
			"name":      clusterName,
			"status":    "scaled",
			"nodeCount": nodeCount,
			"message":   fmt.Sprintf("Cluster '%s' scaled to %d nodes successfully", clusterName, nodeCount),
		}

		if services.GetOutput() == "json" {
			jsonOutput, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			fmt.Println(string(jsonOutput))
		} else {
			fmt.Printf("Cluster '%s' scaled to %d nodes successfully\n", clusterName, nodeCount)
		}

		services.Log("Cluster scale completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
	clusterCmd.AddCommand(clusterStartCmd)
	clusterCmd.AddCommand(clusterStopCmd)
	clusterCmd.AddCommand(clusterScaleCmd)

	clusterCreateCmd.Flags().StringP("provider", "p", "local", "Cloud provider (local, aws, gcp, azure)")
	clusterCreateCmd.Flags().StringP("region", "r", "us-west-2", "Region to create cluster in")
	clusterCreateCmd.Flags().IntP("nodes", "n", 1, "Number of nodes in the cluster")

	clusterListCmd.Flags().StringP("provider", "p", "local", "Cloud provider (local, aws, gcp, azure)")
	
	clusterScaleCmd.Flags().IntP("nodes", "n", 1, "Number of nodes to scale to")
	clusterScaleCmd.MarkFlagRequired("nodes")
}
