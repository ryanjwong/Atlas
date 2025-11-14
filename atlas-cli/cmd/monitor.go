package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/monitoring"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor [cluster-name]",
	Short: "Monitor cluster health and metrics",
	Long:  `Check cluster health status and collect performance metrics.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		providerName, _ := cmd.Flags().GetString("provider")
		region, _ := cmd.Flags().GetString("region") 
		awsProfile, _ := cmd.Flags().GetString("aws-profile")
		
		if providerName == "" {
			providerName = "local"
		}
		
		provider, err := services.GetProvider(providerName, region, awsProfile)
		if err != nil {
			return fmt.Errorf("failed to get provider: %w", err)
		}
		monitor := provider.GetMonitor()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if len(args) == 0 {
			return fmt.Errorf("cluster name is required")
		}

		clusterName := args[0]

		includeMetrics, _ := cmd.Flags().GetBool("metrics")
		watch, _ := cmd.Flags().GetBool("watch")
		
		if watch {
			return monitorWatchMode(ctx, monitor, clusterName, includeMetrics)
		}

		return monitorOneTime(ctx, monitor, clusterName, includeMetrics)
	},
}

func monitorOneTime(ctx context.Context, monitor monitoring.Monitor, clusterName string, includeMetrics bool) error {
	services := GetServices()
	
	services.Log(fmt.Sprintf("Checking health for cluster: %s", clusterName))
	
	healthStatus, err := monitor.CheckClusterHealth(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster health: %w", err)
	}

	if services.GetOutput() == "json" {
		output := map[string]interface{}{
			"health": healthStatus,
		}
		
		if includeMetrics {
			metrics, err := monitor.GetClusterMetrics(ctx, clusterName)
			if err != nil {
				fmt.Printf("Warning: failed to get metrics: %v\n", err)
			} else {
				output["metrics"] = metrics
			}
		}
		
		jsonOutput, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonOutput))
	} else {
		printHealthStatus(healthStatus)
		
		if includeMetrics {
			fmt.Println()
			metrics, err := monitor.GetClusterMetrics(ctx, clusterName)
			if err != nil {
				fmt.Printf("Warning: failed to get metrics: %v\n", err)
			} else {
				printMetrics(metrics)
			}
		}
	}

	return nil
}

func monitorWatchMode(ctx context.Context, monitor monitoring.Monitor, clusterName string, includeMetrics bool) error {
	fmt.Printf("Monitoring cluster '%s' (Press Ctrl+C to exit)\n\n", clusterName)
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			healthStatus, err := monitor.CheckClusterHealth(ctx, clusterName)
			if err != nil {
				fmt.Printf("Health check failed: %v\n", err)
				continue
			}

			fmt.Print("\033[2J\033[H")
			
			fmt.Printf("=== Cluster Monitor: %s ===\n", clusterName)
			fmt.Printf("Last updated: %s\n\n", time.Now().Format("15:04:05"))
			
			printHealthStatus(healthStatus)
			
			if includeMetrics {
				fmt.Println()
				metrics, err := monitor.GetClusterMetrics(ctx, clusterName)
				if err != nil {
					fmt.Printf("Metrics collection failed: %v\n", err)
				} else {
					printMetrics(metrics)
				}
			}
			
			fmt.Println("\n" + strings.Repeat("=", 50))
		}
	}
}

func printHealthStatus(health *monitoring.HealthStatus) {
	fmt.Printf("Overall Status: %s\n", getStatusIcon(string(health.OverallStatus)))
	fmt.Printf("Check Duration: %v\n", health.CheckDuration)
	
	if health.ControlPlane != nil {
		fmt.Println("\n--- Control Plane ---")
		fmt.Printf("API Server:          %s\n", getComponentStatusIcon(health.ControlPlane.APIServer.Status))
		fmt.Printf("Scheduler:           %s\n", getComponentStatusIcon(health.ControlPlane.Scheduler.Status))
		fmt.Printf("Controller Manager:  %s\n", getComponentStatusIcon(health.ControlPlane.ControllerManager.Status))
		fmt.Printf("Etcd:               %s\n", getComponentStatusIcon(health.ControlPlane.Etcd.Status))
	}
	
	if len(health.Nodes) > 0 {
		fmt.Println("\n--- Nodes ---")
		for _, node := range health.Nodes {
			readyIcon := "❌"
			if node.Ready {
				readyIcon = "✅"
			}
			fmt.Printf("%s %s (%s)\n", readyIcon, node.Name, node.Version)
		}
	}
	
	if health.Pods != nil {
		fmt.Println("\n--- Pods ---")
		fmt.Printf("Total: %d | Running: %d | Pending: %d | Failed: %d\n",
			health.Pods.TotalPods, health.Pods.RunningPods, health.Pods.PendingPods, health.Pods.FailedPods)
		
		if len(health.Pods.CriticalPods) > 0 {
			fmt.Println("Critical Pods:")
			for _, pod := range health.Pods.CriticalPods {
				fmt.Printf("  ⚠️  %s/%s (%s)\n", pod.Namespace, pod.Name, pod.Phase)
			}
		}
	}
	
	if health.Services != nil {
		fmt.Printf("\n--- Services ---\n")
		fmt.Printf("Total: %d | Healthy: %d\n", health.Services.TotalServices, health.Services.HealthyServices)
	}
	
	if len(health.Warnings) > 0 {
		fmt.Println("\n--- Warnings ---")
		for _, warning := range health.Warnings {
			fmt.Printf("⚠️  %s\n", warning)
		}
	}
	
	if len(health.Errors) > 0 {
		fmt.Println("\n--- Errors ---")
		for _, error := range health.Errors {
			fmt.Printf("❌ %s\n", error)
		}
	}
}

func printMetrics(metrics *monitoring.ClusterMetrics) {
	fmt.Println("--- Resource Metrics ---")
	
	if len(metrics.NodeMetrics) > 0 {
		fmt.Println("Node Metrics:")
		for _, node := range metrics.NodeMetrics {
			fmt.Printf("  %s: CPU %s (%.1f%%) | Memory %s (%.1f%%)\n",
				node.NodeName, node.CPUUsage.Value, node.CPUUsage.Usage,
				node.MemoryUsage.Value, node.MemoryUsage.Usage)
		}
	}
	
	if metrics.ResourceUsage != nil {
		fmt.Printf("\nCluster Totals:\n")
		fmt.Printf("  CPU Usage: %.1f%%\n", metrics.ResourceUsage.CPUPercentage)
		fmt.Printf("  Memory Usage: %.1f%%\n", metrics.ResourceUsage.MemoryPercentage)
	}
	
	if len(metrics.PodMetrics) > 0 {
		fmt.Printf("\nTop Resource-Consuming Pods:\n")
		maxDisplay := 5
		if len(metrics.PodMetrics) < maxDisplay {
			maxDisplay = len(metrics.PodMetrics)
		}
		
		for i := 0; i < maxDisplay; i++ {
			pod := metrics.PodMetrics[i]
			fmt.Printf("  %s/%s: CPU %s | Memory %s\n",
				pod.Namespace, pod.PodName, pod.CPUUsage.Value, pod.MemoryUsage.Value)
		}
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "healthy":
		return "✅ Healthy"
	case "warning":
		return "⚠️  Warning"
	case "unhealthy":
		return "❌ Unhealthy"
	default:
		return "❓ Unknown"
	}
}

func getComponentStatusIcon(status monitoring.ComponentHealthStatus) string {
	switch status {
	case monitoring.ComponentHealthy:
		return "✅ Healthy"
	case monitoring.ComponentUnhealthy:
		return "❌ Unhealthy"
	default:
		return "❓ Unknown"
	}
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	
	monitorCmd.Flags().BoolP("metrics", "m", false, "Include detailed resource metrics")
	monitorCmd.Flags().BoolP("watch", "w", false, "Watch mode - continuously monitor cluster")
	monitorCmd.Flags().StringP("provider", "p", "local", "Cloud provider (local, aws)")
	monitorCmd.Flags().StringP("region", "r", "", "Region")
	monitorCmd.Flags().String("aws-profile", "", "AWS profile to use (for AWS provider)")
}