package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/logsource"
	"github.com/ryanjwong/Atlas/atlas-cli/pkg/monitoring"
	"github.com/ryanjwong/Atlas/atlas-cli/pkg/providers"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

		configFile, _ := cmd.Flags().GetString("config")
		var config *providers.ClusterConfig

		if configFile != "" {
			var err error
			config, err = loadClusterConfig(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config file: %w", err)
			}
			config.Name = clusterName
		} else {
			region, _ := cmd.Flags().GetString("region")
			nodeCount, _ := cmd.Flags().GetInt("nodes")
			version, _ := cmd.Flags().GetString("version")
			instanceType, _ := cmd.Flags().GetString("instance-type")

			config = &providers.ClusterConfig{
				Name:         clusterName,
				Region:       region,
				NodeCount:    nodeCount,
				Version:      version,
				InstanceType: instanceType,
			}

			enableIngress, _ := cmd.Flags().GetBool("enable-ingress")
			enableLoadBalancer, _ := cmd.Flags().GetBool("enable-load-balancer")
			enableRBAC, _ := cmd.Flags().GetBool("enable-rbac")
			enableNetworkPolicy, _ := cmd.Flags().GetBool("enable-network-policy")
			enableMonitoring, _ := cmd.Flags().GetBool("enable-monitoring")
			apiServerPort, _ := cmd.Flags().GetInt("api-server-port")
			cpuLimit, _ := cmd.Flags().GetString("cpu-limit")
			memoryLimit, _ := cmd.Flags().GetString("memory-limit")

			if enableIngress || enableLoadBalancer || apiServerPort > 0 {
				config.NetworkConfig = &providers.NetworkConfig{}
				if enableIngress {
					config.NetworkConfig.Ingress = &providers.IngressConfig{Enabled: true}
				}
				if enableLoadBalancer {
					config.NetworkConfig.LoadBalancer = &providers.LoadBalancerConfig{Enabled: true}
				}
				if apiServerPort > 0 {
					config.NetworkConfig.APIServerPort = apiServerPort
				}
			}

			if enableRBAC || enableNetworkPolicy {
				config.SecurityConfig = &providers.SecurityConfig{}
				if enableRBAC {
					config.SecurityConfig.RBAC = &providers.RBACConfig{Enabled: true}
				}
				if enableNetworkPolicy {
					config.SecurityConfig.NetworkPolicy = &providers.NetworkPolicyConfig{Enabled: true}
				}
			}

			if enableMonitoring || cpuLimit != "" || memoryLimit != "" {
				config.ResourceConfig = &providers.ResourceConfig{}
				if enableMonitoring {
					config.ResourceConfig.Monitoring = &providers.MonitoringConfig{
						Enabled:    true,
						Prometheus: &providers.PrometheusConfig{Enabled: true},
					}
				}
				if cpuLimit != "" || memoryLimit != "" {
					config.ResourceConfig.Limits = &providers.ResourceLimits{
						CPU:    cpuLimit,
						Memory: memoryLimit,
					}
				}
			}
		}

		provider, _ := cmd.Flags().GetString("provider")
		var p providers.Provider
		switch provider {
		case "local":
			p = services.GetLocalProvider()
		default:
			return fmt.Errorf("unsupported provider: %s", provider)
		}

		if err := p.ValidateConfig(config); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		_, err := p.CreateCluster(context.Background(), config)
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
			p = services.GetLocalProvider()
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

		p := services.GetLocalProvider()
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

		p := services.GetLocalProvider()
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

		p := services.GetLocalProvider()
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

		p := services.GetLocalProvider()
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

var clusterStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show cluster status",
	Long:  `Show current status of a cluster.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		clusterName := args[0]
		p := services.GetLocalProvider()
		actualCluster, err := p.GetCluster(context.Background(), clusterName)
		if err != nil {
			return fmt.Errorf("failed to get cluster status: %w", err)
		}

		if services.GetOutput() == "json" {
			jsonOutput, err := json.MarshalIndent(actualCluster, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal cluster: %w", err)
			}
			fmt.Println(string(jsonOutput))
		} else {
			fmt.Printf("Cluster: %s\n", clusterName)
			fmt.Printf("Provider: %s\n", actualCluster.Provider)
			fmt.Printf("Status: %s\n", actualCluster.Status)
			fmt.Printf("Nodes: %d\n", actualCluster.NodeCount)
			fmt.Printf("Version: %s\n", actualCluster.Version)
			fmt.Printf("Endpoint: %s\n", actualCluster.Endpoint)
		}

		return nil
	},
}


var clusterHistoryCmd = &cobra.Command{
	Use:   "history [name]",
	Short: "Show cluster operation history",
	Long:  `Show the history of operations performed on a cluster from minikube's audit logs.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		clusterName := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		
		provider := services.GetLocalProvider()
		logSource := provider.GetLogSource()
		
		operationHistory, err := logSource.GetClusterHistory(context.Background(), clusterName, limit)
		if err != nil {
			return fmt.Errorf("failed to get cluster history: %w", err)
		}

		if services.GetOutput() == "json" {
			jsonOutput, err := json.MarshalIndent(operationHistory, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal operation history: %w", err)
			}
			fmt.Println(string(jsonOutput))
			return nil
		}

		if len(operationHistory) == 0 {
			fmt.Printf("No operations found for cluster '%s'\n", clusterName)
			return nil
		}

		fmt.Printf("Operation History for '%s' (%d operations):\n\n", clusterName, len(operationHistory))
		fmt.Printf("%-20s %-8s %-10s %-12s %-12s\n", "STARTED", "TYPE", "STATUS", "USER", "DURATION")
		fmt.Printf("%-20s %-8s %-10s %-12s %-12s\n", "----", "----", "----", "----", "----")

		for _, op := range operationHistory {
			started := op.StartedAt.Format("Jan 02 15:04:05")
			statusColor := getStatusColor(op.OperationStatus)
			
			duration := "-"
			if op.DurationMS != nil {
				if *op.DurationMS < 1000 {
					duration = fmt.Sprintf("%.0fms", *op.DurationMS)
				} else {
					duration = fmt.Sprintf("%.1fs", *op.DurationMS/1000)
				}
			}
			
			fmt.Printf("%-20s %-8s %s%-10s%s %-12s %-12s\n",
				started,
				string(op.OperationType),
				statusColor,
				string(op.OperationStatus),
				"\033[0m", 
				truncateString(op.UserID, 12),
				duration)
		}

		return nil
	},
}

var clusterWatchCmd = &cobra.Command{
	Use:   "watch [name]",
	Short: "Watch cluster health in real-time",
	Long:  `Monitor cluster health and resource usage in real-time with automatic updates.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := GetServices()
		if services == nil {
			return fmt.Errorf("services not initialized")
		}

		clusterName := args[0]
		provider := services.GetLocalProvider()
		monitor := provider.GetMonitor()
		
		includeMetrics, _ := cmd.Flags().GetBool("metrics")
		interval, _ := cmd.Flags().GetInt("interval")
		
		return watchCluster(monitor, clusterName, includeMetrics, interval)
	},
}

func loadClusterConfig(configFile string) (*providers.ClusterConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config providers.ClusterConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &config, nil
}

func watchCluster(monitor monitoring.Monitor, clusterName string, includeMetrics bool, intervalSecs int) error {
	fmt.Printf("Watching cluster '%s' (Press Ctrl+C to exit)\n\n", clusterName)
	
	interval := time.Duration(intervalSecs) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	ctx := context.Background()

	for {
		healthStatus, err := monitor.CheckClusterHealth(ctx, clusterName)
		if err != nil {
			fmt.Printf("Health check failed: %v\n", err)
			time.Sleep(interval)
			continue
		}

		fmt.Print("\033[2J\033[H")
		
		fmt.Printf("=== Cluster Monitor: %s ===\n", clusterName)
		fmt.Printf("Last updated: %s\n\n", time.Now().Format("15:04:05"))
		
		printClusterHealthStatus(healthStatus)
		
		if includeMetrics {
			fmt.Println()
			metrics, err := monitor.GetClusterMetrics(ctx, clusterName)
			if err != nil {
				fmt.Printf("Metrics collection failed: %v\n", err)
			} else {
				printClusterMetrics(metrics)
			}
		}
		
		fmt.Println("\n" + strings.Repeat("=", 50))
		
		select {
		case <-ticker.C:
			continue
		}
	}
}

func printClusterHealthStatus(health *monitoring.HealthStatus) {
	fmt.Printf("Overall Status: %s\n", getStatusDisplayIcon(string(health.OverallStatus)))
	fmt.Printf("Check Duration: %v\n", health.CheckDuration)
	
	if health.ControlPlane != nil {
		fmt.Println("\n--- Control Plane ---")
		fmt.Printf("API Server:          %s\n", getComponentStatusDisplayIcon(health.ControlPlane.APIServer.Status))
		fmt.Printf("Scheduler:           %s\n", getComponentStatusDisplayIcon(health.ControlPlane.Scheduler.Status))
		fmt.Printf("Controller Manager:  %s\n", getComponentStatusDisplayIcon(health.ControlPlane.ControllerManager.Status))
		fmt.Printf("Etcd:               %s\n", getComponentStatusDisplayIcon(health.ControlPlane.Etcd.Status))
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

func printClusterMetrics(metrics *monitoring.ClusterMetrics) {
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

func getStatusDisplayIcon(status string) string {
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

func getComponentStatusDisplayIcon(status monitoring.ComponentHealthStatus) string {
	switch status {
	case monitoring.ComponentHealthy:
		return "✅ Healthy"
	case monitoring.ComponentUnhealthy:
		return "❌ Unhealthy"
	default:
		return "❓ Unknown"
	}
}

var clusterGenerateConfigCmd = &cobra.Command{
	Use:   "generate-config [name]",
	Short: "Generate a sample configuration file",
	Long:  `Generate a sample YAML configuration file for creating clusters with advanced options.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := args[0]
		outputFile, _ := cmd.Flags().GetString("output")

		sampleConfig := &providers.ClusterConfig{
			Name:         clusterName,
			Region:       "local",
			Version:      "v1.31.0",
			NodeCount:    2,
			InstanceType: "standard",
			NetworkConfig: &providers.NetworkConfig{
				PodCIDR:       "10.244.0.0/16",
				ServiceCIDR:   "10.96.0.0/12",
				APIServerPort: 8443,
				NetworkPlugin: "auto",
				Ingress: &providers.IngressConfig{
					Enabled:    true,
					Controller: "nginx",
				},
				LoadBalancer: &providers.LoadBalancerConfig{
					Enabled: true,
					Type:    "metallb",
				},
			},
			SecurityConfig: &providers.SecurityConfig{
				RBAC: &providers.RBACConfig{
					Enabled: true,
				},
				NetworkPolicy: &providers.NetworkPolicyConfig{
					Enabled:       true,
					DefaultPolicy: "deny-all",
				},
				AuditLogging: &providers.AuditConfig{
					Enabled:  true,
					LogLevel: "2",
				},
				ImageSecurity: &providers.ImageSecurityConfig{
					ScanEnabled:            true,
					VulnerabilityThreshold: "medium",
				},
			},
			ResourceConfig: &providers.ResourceConfig{
				Limits: &providers.ResourceLimits{
					CPU:    "4",
					Memory: "8Gi",
				},
				Quotas: &providers.ResourceQuotas{
					DefaultQuota: &providers.NamespaceQuota{
						CPU:     "2",
						Memory:  "4Gi",
						Storage: "10Gi",
						Pods:    10,
						PVCs:    5,
					},
				},
				AutoScaling: &providers.AutoScalingConfig{
					Enabled:   true,
					MinNodes:  1,
					MaxNodes:  5,
					TargetCPU: 70,
				},
				Monitoring: &providers.MonitoringConfig{
					Enabled: true,
					Prometheus: &providers.PrometheusConfig{
						Enabled:        true,
						Retention:      "15d",
						StorageSize:    "5Gi",
						ScrapeInterval: "30s",
					},
					Grafana: &providers.GrafanaConfig{
						Enabled:     true,
						AdminUser:   "admin",
						Persistence: true,
					},
				},
				Storage: &providers.StorageConfig{
					DefaultStorageClass: "hostpath",
					VolumeExpansion:     true,
					SnapshotController:  true,
					StorageClasses: []providers.StorageClassConfig{
						{
							Name:        "fast",
							Provisioner: "hostpath",
							Default:     false,
						},
					},
				},
			},
			Tags: map[string]string{
				"environment": "development",
				"team":        "platform",
				"purpose":     "testing",
			},
		}

		yamlData, err := yaml.Marshal(sampleConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal config to YAML: %w", err)
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, yamlData, 0644); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}
			fmt.Printf("Sample configuration written to %s\n", outputFile)
		} else {
			fmt.Print(string(yamlData))
		}

		return nil
	},
}

func getStatusColor(status logsource.OperationStatus) string {
	switch status {
	case logsource.OpStatusCompleted:
		return "\033[32m"
	case logsource.OpStatusRunning:
		return "\033[33m"
	case logsource.OpStatusFailed:
		return "\033[31m"
	default:
		return "\033[37m"
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
	clusterCmd.AddCommand(clusterStartCmd)
	clusterCmd.AddCommand(clusterStopCmd)
	clusterCmd.AddCommand(clusterScaleCmd)
	clusterCmd.AddCommand(clusterGenerateConfigCmd)
	clusterCmd.AddCommand(clusterStatusCmd)
	clusterCmd.AddCommand(clusterHistoryCmd)
	clusterCmd.AddCommand(clusterWatchCmd)

	clusterCreateCmd.Flags().StringP("provider", "p", "local", "Cloud provider (local, aws, gcp, azure)")
	clusterCreateCmd.Flags().StringP("region", "r", "local", "Region to create cluster in")
	clusterCreateCmd.Flags().IntP("nodes", "n", 1, "Number of nodes in the cluster")
	clusterCreateCmd.Flags().StringP("version", "k", "", "Kubernetes version")
	clusterCreateCmd.Flags().String("instance-type", "", "Instance type for nodes")
	clusterCreateCmd.Flags().StringP("config", "c", "", "Path to cluster configuration YAML file")

	clusterCreateCmd.Flags().Bool("enable-ingress", false, "Enable ingress controller")
	clusterCreateCmd.Flags().Bool("enable-load-balancer", false, "Enable load balancer")
	clusterCreateCmd.Flags().Bool("enable-rbac", false, "Enable RBAC")
	clusterCreateCmd.Flags().Bool("enable-network-policy", false, "Enable network policies")
	clusterCreateCmd.Flags().Bool("enable-monitoring", false, "Enable monitoring stack")
	clusterCreateCmd.Flags().Int("api-server-port", 0, "API server port (0 for default)")
	clusterCreateCmd.Flags().String("cpu-limit", "", "CPU limit per node (e.g., '4', '2.5')")
	clusterCreateCmd.Flags().String("memory-limit", "", "Memory limit per node (e.g., '8Gi', '4096Mi')")

	clusterListCmd.Flags().StringP("provider", "p", "local", "Cloud provider (local, aws, gcp, azure)")

	clusterScaleCmd.Flags().IntP("nodes", "n", 1, "Number of nodes to scale to")
	clusterScaleCmd.MarkFlagRequired("nodes")

	clusterGenerateConfigCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")

	clusterHistoryCmd.Flags().IntP("limit", "l", 50, "Number of operations to display")
	
	clusterWatchCmd.Flags().BoolP("metrics", "m", false, "Include detailed resource metrics")
	clusterWatchCmd.Flags().IntP("interval", "i", 5, "Update interval in seconds")
}
