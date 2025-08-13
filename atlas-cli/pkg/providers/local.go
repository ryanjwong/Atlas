package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/logsource"
	"github.com/ryanjwong/Atlas/atlas-cli/pkg/monitoring"
)

// LocalProvider implements Provider for local minikube clusters
type LocalProvider struct {
	logSource logsource.LogSource
	monitor   monitoring.Monitor
}

// NewLocalProvider creates a new local provider
func NewLocalProvider() *LocalProvider {
	return &LocalProvider{
		logSource: logsource.NewMinikubeLogSource(),
		monitor:   monitoring.NewMinikubeMonitor(),
	}
}

type MinikubeProfilesResponse struct {
	Invalid []interface{} `json:"invalid"`
	Valid   []Profile     `json:"valid"`
}

type Profile struct {
	Name string `json:"Name"`
}

// GetLogSource returns the log source for reading operation history
func (l *LocalProvider) GetLogSource() logsource.LogSource {
	return l.logSource
}

// GetProviderName returns the name of this provider
func (l *LocalProvider) GetProviderName() string {
	return "local"
}

// GetSupportedRegions returns the list of supported regions for the local provider
func (l *LocalProvider) GetSupportedRegions() []string {
	return []string{"local"}
}

// GetSupportedVersions returns the list of supported Kubernetes versions for the local provider
func (l *LocalProvider) GetSupportedVersions() []string {
	return []string{"v1.31.0", "v1.30.0", "v1.29.0", "v1.28.0", "v1.27.0"}
}

// CreateCluster creates a new minikube cluster with the specified configuration
func (l *LocalProvider) CreateCluster(ctx context.Context, config *ClusterConfig) (*Cluster, error) {
	if err := l.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	args := []string{"start", "-p", config.Name}

	if config.Version != "" {
		args = append(args, "--kubernetes-version="+config.Version)
	}

	if config.NodeCount > 0 {
		args = append(args, "--nodes="+strconv.Itoa(config.NodeCount))
	}

	if config.NetworkConfig != nil {
		if config.NetworkConfig.PodCIDR != "" {
			args = append(args, "--extra-config", "kubeadm.pod-network-cidr="+config.NetworkConfig.PodCIDR)
		}
		if config.NetworkConfig.ServiceCIDR != "" {
			args = append(args, "--service-cluster-ip-range", config.NetworkConfig.ServiceCIDR)
		}
		if config.NetworkConfig.APIServerPort > 0 {
			args = append(args, "--apiserver-port", strconv.Itoa(config.NetworkConfig.APIServerPort))
		}
		if config.NetworkConfig.NetworkPlugin != "" && config.NetworkConfig.NetworkPlugin != "auto" {
			args = append(args, "--cni", config.NetworkConfig.NetworkPlugin)
		}
	}

	if config.SecurityConfig != nil {
		if config.SecurityConfig.RBAC != nil && config.SecurityConfig.RBAC.Enabled {
			args = append(args, "--extra-config", "apiserver.authorization-mode=RBAC")
		}
		if config.SecurityConfig.AuditLogging != nil && config.SecurityConfig.AuditLogging.Enabled {
			args = append(args, "--extra-config", "apiserver.audit-log-path=/tmp/audit.log")
			if config.SecurityConfig.AuditLogging.LogLevel != "" {
				args = append(args, "--extra-config", "apiserver.v="+config.SecurityConfig.AuditLogging.LogLevel)
			}
		}
	}

	if config.ResourceConfig != nil {
		if config.ResourceConfig.Limits != nil {
			if config.ResourceConfig.Limits.CPU != "" {
				args = append(args, "--cpus", config.ResourceConfig.Limits.CPU)
			}
			if config.ResourceConfig.Limits.Memory != "" {
				args = append(args, "--memory", config.ResourceConfig.Limits.Memory)
			}
		}
	}

	cmd := exec.CommandContext(ctx, "minikube", args...)
	fmt.Println("Creating minikube cluster...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster %s: %w\nOutput: %s", config.Name, err, string(output))
	}

	if err := l.applyPostCreateConfigs(ctx, config); err != nil {
		fmt.Printf("Warning: failed to apply some post-create configurations: %v\n", err)
	}

	fmt.Printf("Successfully created cluster: %s\n", config.Name)
	return l.GetCluster(ctx, config.Name)
}

// DeleteCluster deletes a minikube cluster by name
func (l *LocalProvider) DeleteCluster(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "minikube", "delete", "-p", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete cluster %s: %w\nOutput: %s", name, err, string(output))
	}
	return nil
}

// StartCluster starts a stopped minikube cluster
func (l *LocalProvider) StartCluster(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "minikube", "start", "-p", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start cluster %s: %w\nOutput: %s", name, err, string(output))
	}
	return nil
}

// StopCluster stops a running minikube cluster
func (l *LocalProvider) StopCluster(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "minikube", "stop", "-p", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop cluster %s: %w\nOutput: %s", name, err, string(output))
	}
	return nil
}

// ScaleCluster scales a minikube cluster to the specified number of nodes
func (l *LocalProvider) ScaleCluster(ctx context.Context, name string, nodeCount int) error {
	if nodeCount <= 0 {
		return fmt.Errorf("node count must be positive")
	}
	if nodeCount > 10 {
		return fmt.Errorf("node count cannot exceed 10 for local provider")
	}

	currentCluster, err := l.GetCluster(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get current cluster info: %w", err)
	}

	if currentCluster.NodeCount == nodeCount {
		return nil 
	}

	if nodeCount > currentCluster.NodeCount {
		for i := currentCluster.NodeCount; i < nodeCount; i++ {
			cmd := exec.CommandContext(ctx, "minikube", "node", "add", "-p", name)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to add node to cluster %s: %w\nOutput: %s", name, err, string(output))
			}
		}
	} else {
		for i := currentCluster.NodeCount; i > nodeCount; i-- {
			cmd := exec.CommandContext(ctx, "minikube", "node", "delete", fmt.Sprintf("%s-m%02d", name, i-1), "-p", name)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to remove node from cluster %s: %w\nOutput: %s", name, err, string(output))
			}
		}
	}

	return nil
}

// GetCluster retrieves information about a minikube cluster
func (l *LocalProvider) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	cmd := exec.CommandContext(ctx, "minikube", "status", "-p", name)
	output, err := cmd.CombinedOutput()
	statusStr := string(output)

	var status ClusterStatus
	if strings.Contains(statusStr, "Running") {
		status = ClusterStatusRunning
	} else if strings.Contains(statusStr, "Stopped") {
		status = ClusterStatusStopped
	} else if err != nil {
		if strings.Contains(err.Error(), "exit status 7") || strings.Contains(statusStr, "does not exist") {
			return nil, fmt.Errorf("cluster %s does not exist", name)
		}
		status = ClusterStatusError
	} else {
		status = ClusterStatusError
	}

	cmd = exec.CommandContext(ctx, "minikube", "ip", "-p", name)
	ipOutput, err := cmd.CombinedOutput()
	var endpoint string
	if err == nil {
		endpoint = strings.TrimSpace(string(ipOutput))
	}

	var version string
	var nodeCount int = 1

	cmd = exec.CommandContext(ctx, "minikube", "profile", "list")
	profileOutput, err := cmd.CombinedOutput()
	if err == nil {
		lines := strings.Split(string(profileOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, name) {
				fields := strings.Fields(line)
				if len(fields) >= 8 {
					version = fields[5]
					if nodeCountStr := fields[7]; nodeCountStr != "" {
						if count, parseErr := strconv.Atoi(nodeCountStr); parseErr == nil {
							nodeCount = count
						}
					}
				}
				break
			}
		}
	}

	if version == "" && status == ClusterStatusRunning {
		cmd = exec.CommandContext(ctx, "minikube", "kubectl", "-p", name, "--", "version", "--client=false", "--output=yaml")
		versionOutput, err := cmd.CombinedOutput()
		if err == nil {
			lines := strings.Split(string(versionOutput), "\n")
			for _, line := range lines {
				if strings.Contains(line, "gitVersion:") {
					parts := strings.Split(line, ":")
					if len(parts) >= 2 {
						version = strings.TrimSpace(strings.Trim(parts[1], " \""))
					}
					break
				}
			}
		}
	}

	if status == ClusterStatusRunning {
		cmd = exec.CommandContext(ctx, "minikube", "kubectl", "-p", name, "--", "get", "nodes", "--no-headers")
		nodesOutput, err := cmd.CombinedOutput()
		if err == nil {
			nodeLines := strings.Split(strings.TrimSpace(string(nodesOutput)), "\n")
			if len(nodeLines) > 0 && nodeLines[0] != "" {
				nodeCount = len(nodeLines)
			}
		}
	}

	return &Cluster{
		Name:      name,
		Provider:  "local",
		Region:    "local",
		Version:   version,
		Status:    status,
		NodeCount: nodeCount,
		Endpoint:  endpoint,
		CreatedAt: time.Now(), // We could get this from log source if needed
		UpdatedAt: time.Now(),
		Tags:      make(map[string]string),
	}, nil
}

// ListClusters lists all minikube clusters managed by this provider
func (l *LocalProvider) ListClusters(ctx context.Context) ([]*Cluster, error) {
	cmd := exec.CommandContext(ctx, "minikube", "profile", "list", "-o=json")
	var profiles MinikubeProfilesResponse

	profileOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error getting profiles: %w", err)
	}
	
	if err := json.Unmarshal(profileOutput, &profiles); err != nil {
		return nil, fmt.Errorf("error unmarshaling profiles: %w", err)
	}

	var clusters []*Cluster
	for _, profile := range profiles.Valid {
		cluster, err := l.GetCluster(ctx, profile.Name)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return nil, fmt.Errorf("error getting cluster %s: %w", profile.Name, err)
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// ValidateConfig validates the cluster configuration for the local provider
func (l *LocalProvider) ValidateConfig(config *ClusterConfig) error {
	if config.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	cmd := exec.Command("minikube", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("minikube is not installed or not in PATH")
	}

	if strings.Contains(config.Name, " ") {
		return fmt.Errorf("cluster name cannot contain spaces")
	}

	if config.NodeCount < 0 {
		return fmt.Errorf("node count cannot be negative")
	}
	if config.NodeCount > 10 {
		return fmt.Errorf("node count cannot exceed 10 for local provider")
	}

	if err := l.validateNetworkConfig(config.NetworkConfig); err != nil {
		return fmt.Errorf("invalid network configuration: %w", err)
	}

	if err := l.validateSecurityConfig(config.SecurityConfig); err != nil {
		return fmt.Errorf("invalid security configuration: %w", err)
	}

	if err := l.validateResourceConfig(config.ResourceConfig); err != nil {
		return fmt.Errorf("invalid resource configuration: %w", err)
	}

	return nil
}

// applyPostCreateConfigs applies post-creation configurations like networking, security, and resources
func (l *LocalProvider) applyPostCreateConfigs(ctx context.Context, config *ClusterConfig) error {
	if config.NetworkConfig != nil {
		if err := l.applyNetworkConfig(ctx, config.Name, config.NetworkConfig); err != nil {
			return fmt.Errorf("failed to apply network config: %w", err)
		}
	}

	if config.SecurityConfig != nil {
		if err := l.applySecurityConfig(ctx, config.Name, config.SecurityConfig); err != nil {
			return fmt.Errorf("failed to apply security config: %w", err)
		}
	}

	if config.ResourceConfig != nil {
		if err := l.applyResourceConfig(ctx, config.Name, config.ResourceConfig); err != nil {
			return fmt.Errorf("failed to apply resource config: %w", err)
		}
	}

	return nil
}

// applyNetworkConfig applies network configuration including ingress and load balancer settings
func (l *LocalProvider) applyNetworkConfig(ctx context.Context, clusterName string, netConfig *NetworkConfig) error {
	if netConfig.Ingress != nil && netConfig.Ingress.Enabled {
		cmd := exec.CommandContext(ctx, "minikube", "addons", "enable", "ingress", "-p", clusterName)
		if _, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to enable ingress addon: %w", err)
		}
		fmt.Printf("Enabled ingress controller for cluster %s\n", clusterName)
	}

	if netConfig.LoadBalancer != nil && netConfig.LoadBalancer.Enabled {
		cmd := exec.CommandContext(ctx, "minikube", "addons", "enable", "metallb", "-p", clusterName)
		if _, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to enable metallb addon: %w", err)
		}
		fmt.Printf("Enabled MetalLB load balancer for cluster %s\n", clusterName)
	}

	return nil
}

// applySecurityConfig applies security configuration including network policies
func (l *LocalProvider) applySecurityConfig(ctx context.Context, clusterName string, secConfig *SecurityConfig) error {
	if secConfig.NetworkPolicy != nil && secConfig.NetworkPolicy.Enabled {
		networkPolicyYAML := `
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
`
		if err := l.applyKubernetesResource(ctx, clusterName, networkPolicyYAML); err != nil {
			return fmt.Errorf("failed to apply network policy: %w", err)
		}
		fmt.Printf("Applied default network policy for cluster %s\n", clusterName)
	}

	return nil
}

// applyResourceConfig applies resource configuration including monitoring and storage settings
func (l *LocalProvider) applyResourceConfig(ctx context.Context, clusterName string, resConfig *ResourceConfig) error {
	if resConfig.Monitoring != nil && resConfig.Monitoring.Enabled {
		if resConfig.Monitoring.Prometheus != nil && resConfig.Monitoring.Prometheus.Enabled {
			cmd := exec.CommandContext(ctx, "minikube", "addons", "enable", "metrics-server", "-p", clusterName)
			if _, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to enable metrics-server addon: %w", err)
			}
			fmt.Printf("Enabled metrics-server for cluster %s\n", clusterName)
		}
	}

	if resConfig.Storage != nil {
		if resConfig.Storage.DefaultStorageClass != "" {
			cmd := exec.CommandContext(ctx, "minikube", "addons", "enable", "default-storageclass", "-p", clusterName)
			if _, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to enable default storageclass: %w", err)
			}
			fmt.Printf("Enabled default storage class for cluster %s\n", clusterName)
		}
	}

	return nil
}

// applyKubernetesResource applies a YAML resource to the minikube cluster
func (l *LocalProvider) applyKubernetesResource(ctx context.Context, clusterName, resourceYAML string) error {
	cmd := exec.CommandContext(ctx, "minikube", "kubectl", "-p", clusterName, "--", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(resourceYAML)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to apply kubernetes resource: %w", err)
	}
	return nil
}

// validateNetworkConfig validates network configuration parameters
func (l *LocalProvider) validateNetworkConfig(netConfig *NetworkConfig) error {
	if netConfig == nil {
		return nil
	}

	if netConfig.APIServerPort > 0 && (netConfig.APIServerPort < 1024 || netConfig.APIServerPort > 65535) {
		return fmt.Errorf("API server port must be between 1024 and 65535")
	}

	if netConfig.NetworkPlugin != "" {
		validPlugins := []string{"bridge", "flannel", "calico", "auto"}
		isValid := false
		for _, plugin := range validPlugins {
			if netConfig.NetworkPlugin == plugin {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid network plugin: %s. Valid options: %v", netConfig.NetworkPlugin, validPlugins)
		}
	}

	for _, portMap := range netConfig.ExtraPortMaps {
		if portMap.HostPort <= 0 || portMap.ContainerPort <= 0 {
			return fmt.Errorf("port mappings must have positive port numbers")
		}
		if portMap.Protocol != "" && portMap.Protocol != "tcp" && portMap.Protocol != "udp" {
			return fmt.Errorf("invalid protocol: %s. Valid options: tcp, udp", portMap.Protocol)
		}
	}

	if netConfig.Ingress != nil && netConfig.Ingress.Controller != "" {
		validControllers := []string{"nginx", "traefik", "haproxy"}
		isValid := false
		for _, controller := range validControllers {
			if netConfig.Ingress.Controller == controller {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid ingress controller: %s. Valid options: %v", netConfig.Ingress.Controller, validControllers)
		}
	}

	return nil
}

// validateSecurityConfig validates security configuration parameters
func (l *LocalProvider) validateSecurityConfig(secConfig *SecurityConfig) error {
	if secConfig == nil {
		return nil
	}

	if secConfig.AuthenticationMode != "" {
		validModes := []string{"RBAC", "ABAC", "Node", "Webhook"}
		isValid := false
		for _, mode := range validModes {
			if secConfig.AuthenticationMode == mode {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid authentication mode: %s. Valid options: %v", secConfig.AuthenticationMode, validModes)
		}
	}

	if secConfig.AuditLogging != nil && secConfig.AuditLogging.LogLevel != "" {
		validLevels := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
		isValid := false
		for _, level := range validLevels {
			if secConfig.AuditLogging.LogLevel == level {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid audit log level: %s. Valid options: 1-10", secConfig.AuditLogging.LogLevel)
		}
	}

	if secConfig.ImageSecurity != nil && secConfig.ImageSecurity.VulnerabilityThreshold != "" {
		validThresholds := []string{"low", "medium", "high", "critical"}
		isValid := false
		for _, threshold := range validThresholds {
			if secConfig.ImageSecurity.VulnerabilityThreshold == threshold {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid vulnerability threshold: %s. Valid options: %v", secConfig.ImageSecurity.VulnerabilityThreshold, validThresholds)
		}
	}

	return nil
}

// validateResourceConfig validates resource configuration parameters
func (l *LocalProvider) validateResourceConfig(resConfig *ResourceConfig) error {
	if resConfig == nil {
		return nil
	}

	if resConfig.AutoScaling != nil {
		if resConfig.AutoScaling.MinNodes < 1 {
			return fmt.Errorf("minimum nodes must be at least 1")
		}
		if resConfig.AutoScaling.MaxNodes > 10 {
			return fmt.Errorf("maximum nodes cannot exceed 10 for local provider")
		}
		if resConfig.AutoScaling.MinNodes > resConfig.AutoScaling.MaxNodes {
			return fmt.Errorf("minimum nodes cannot be greater than maximum nodes")
		}
		if resConfig.AutoScaling.TargetCPU > 0 && (resConfig.AutoScaling.TargetCPU < 10 || resConfig.AutoScaling.TargetCPU > 90) {
			return fmt.Errorf("target CPU must be between 10 and 90 percent")
		}
	}

	if resConfig.Storage != nil {
		for _, sc := range resConfig.Storage.StorageClasses {
			if sc.Name == "" || sc.Provisioner == "" {
				return fmt.Errorf("storage class name and provisioner are required")
			}
			validProvisioners := []string{"hostpath", "local", "nfs"}
			isValid := false
			for _, provisioner := range validProvisioners {
				if sc.Provisioner == provisioner {
					isValid = true
					break
				}
			}
			if !isValid {
				return fmt.Errorf("invalid storage provisioner: %s. Valid options for local provider: %v", sc.Provisioner, validProvisioners)
			}
		}
	}

	return nil
}

// GetMonitor returns the monitor for health checks and metrics collection
func (l *LocalProvider) GetMonitor() monitoring.Monitor {
	return l.monitor
}

// HealthCheck performs a health check on the specified cluster
func (l *LocalProvider) HealthCheck(ctx context.Context, clusterName string) (*monitoring.HealthStatus, error) {
	return l.monitor.CheckClusterHealth(ctx, clusterName)
}

// Ensure LocalProvider implements Provider interface
var _ Provider = (*LocalProvider)(nil)