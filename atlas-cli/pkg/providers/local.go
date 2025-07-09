package providers

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type LocalProvider struct {
}

// StartCluster implements Provider.
func (l *LocalProvider) StartCluster(ctx context.Context, name string) error {
	// Check if cluster exists
	cmd := exec.Command("minikube", "status", "-p", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s does not exist: %s", name, err)
	}

	// Check if already running
	if strings.Contains(string(output), "Running") {
		return nil // Already running
	}

	// Start the cluster
	cmd = exec.Command("minikube", "start", "-p", name)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start cluster %s: %s", name, err)
	}

	return nil
}

// StopCluster implements Provider.
func (l *LocalProvider) StopCluster(ctx context.Context, name string) error {
	// Check if cluster exists
	cmd := exec.Command("minikube", "status", "-p", name)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s does not exist: %s", name, err)
	}

	// Stop the cluster
	cmd = exec.Command("minikube", "stop", "-p", name)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop cluster %s: %s", name, err)
	}

	return nil
}

// CreateCluster implements CloudProvider.
func (l *LocalProvider) CreateCluster(ctx context.Context, config *ClusterConfig) (*Cluster, error) {
	// Build minikube start command with proper arguments
	args := []string{"start", "-p", config.Name}
	
	// Add Kubernetes version if specified
	if config.Version != "" {
		args = append(args, "--kubernetes-version="+config.Version)
	}
	
	// Add node count if specified (minikube supports multi-node with --nodes flag)
	if config.NodeCount > 0 {
		args = append(args, "--nodes="+strconv.Itoa(config.NodeCount))
	}
	
	cmd := exec.Command("minikube", args...)
	fmt.Println("creating minikube cluster...")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to start minikube: %s", err)
	}

	// Get the created cluster info
	return l.GetCluster(ctx, config.Name)
}

// DeleteCluster implements CloudProvider.
func (l *LocalProvider) DeleteCluster(ctx context.Context, name string) error {
	// Delete the cluster
	cmd := exec.Command("minikube", "delete", "-p", name)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete cluster %s: %s", name, err)
	}

	return nil
}

// GetCluster implements CloudProvider.
func (l *LocalProvider) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	// Get cluster status
	cmd := exec.Command("minikube", "status", "-p", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cluster %s not found: %s", name, err)
	}

	// Parse status
	statusStr := string(output)
	var status ClusterStatus
	if strings.Contains(statusStr, "Running") {
		status = ClusterStatusRunning
	} else if strings.Contains(statusStr, "Stopped") {
		status = ClusterStatusStopped
	} else {
		status = ClusterStatusError
	}

	// Get cluster IP
	cmd = exec.Command("minikube", "ip", "-p", name)
	ipOutput, err := cmd.CombinedOutput()
	var endpoint string
	if err == nil {
		endpoint = strings.TrimSpace(string(ipOutput))
	}

	// Get version and node count from profile list
	var version string
	var nodeCount int = 1 // default
	
	cmd = exec.Command("minikube", "profile", "list")
	profileOutput, err := cmd.CombinedOutput()
	if err == nil {
		lines := strings.Split(string(profileOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, name) {
				fields := strings.Fields(line)
				if len(fields) >= 8 {
					// Fields: Profile, VM Driver, Runtime, IP, Port, Version, Status, Nodes
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

	// If profile list didn't work and cluster is running, try kubectl
	if version == "" && status == ClusterStatusRunning {
		cmd = exec.Command("minikube", "kubectl", "-p", name, "--", "version", "--client=false", "--output=yaml")
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

	// Get actual node count if cluster is running
	if status == ClusterStatusRunning {
		cmd = exec.Command("minikube", "kubectl", "-p", name, "--", "get", "nodes", "--no-headers")
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
		CreatedAt: time.Now(), // Cannot determine actual creation time
		UpdatedAt: time.Now(),
		Tags:      make(map[string]string),
	}, nil
}

// GetProviderName implements CloudProvider.
func (l *LocalProvider) GetProviderName() string {
	return "local"
}

// GetSupportedRegions implements CloudProvider.
func (l *LocalProvider) GetSupportedRegions() []string {
	return []string{"local"}
}

// GetSupportedVersions implements CloudProvider.
func (l *LocalProvider) GetSupportedVersions() []string {
	return []string{"v1.31.0", "v1.30.0", "v1.29.0", "v1.28.0", "v1.27.0"}
}

// ScaleCluster implements CloudProvider.
func (l *LocalProvider) ScaleCluster(ctx context.Context, name string, nodeCount int) error {
	if nodeCount <= 0 {
		return fmt.Errorf("node count must be positive")
	}
	if nodeCount > 10 {
		return fmt.Errorf("node count cannot exceed 10 for local provider")
	}

	// Get current node count
	currentCluster, err := l.GetCluster(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get current cluster info: %w", err)
	}

	if currentCluster.NodeCount == nodeCount {
		return nil // Already at desired count
	}

	if nodeCount > currentCluster.NodeCount {
		// Add nodes
		for i := currentCluster.NodeCount; i < nodeCount; i++ {
			cmd := exec.Command("minikube", "node", "add", "-p", name)
			if _, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to add node to cluster %s: %s", name, err)
			}
		}
	} else {
		// Remove nodes
		for i := currentCluster.NodeCount; i > nodeCount; i-- {
			cmd := exec.Command("minikube", "node", "delete", fmt.Sprintf("%s-m%02d", name, i-1), "-p", name)
			if _, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to remove node from cluster %s: %s", name, err)
			}
		}
	}

	return nil
}

// UpdateCluster implements CloudProvider.
func (l *LocalProvider) UpdateCluster(ctx context.Context, name string, config *ClusterConfig) (*Cluster, error) {
	// For local provider, updating means stopping and restarting with new config
	if err := l.StopCluster(ctx, name); err != nil {
		return nil, fmt.Errorf("failed to stop cluster for update: %w", err)
	}

	// Delete the old cluster
	if err := l.DeleteCluster(ctx, name); err != nil {
		return nil, fmt.Errorf("failed to delete cluster for update: %w", err)
	}

	// Create new cluster with updated config
	return l.CreateCluster(ctx, config)
}

// ValidateConfig implements CloudProvider.
func (l *LocalProvider) ValidateConfig(config *ClusterConfig) error {
	if config.Name == "" {
		return fmt.Errorf("cluster name is required")
	}

	// Check if minikube is installed
	cmd := exec.Command("minikube", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("minikube is not installed or not in PATH")
	}

	// Validate cluster name format
	if strings.Contains(config.Name, " ") {
		return fmt.Errorf("cluster name cannot contain spaces")
	}

	// Validate node count (minikube supports multi-node clusters)
	if config.NodeCount < 0 {
		return fmt.Errorf("node count cannot be negative")
	}
	if config.NodeCount > 10 {
		return fmt.Errorf("node count cannot exceed 10 for local provider")
	}

	return nil
}

var _ Provider = (*LocalProvider)(nil)
