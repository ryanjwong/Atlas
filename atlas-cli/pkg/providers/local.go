package providers

import (
	"context"
	"fmt"
	"os/exec"
)

type LocalProvider struct {
}

// StartCluster implements Provider.
func (l *LocalProvider) StartCluster(ctx context.Context, name string) error {
	panic("unimplemented")
}

// StopCluster implements Provider.
func (l *LocalProvider) StopCluster(ctx context.Context, name string) error {
	panic("unimplemented")
}

// CreateCluster implements CloudProvider.
func (l *LocalProvider) CreateCluster(ctx context.Context, config *ClusterConfig) (*Cluster, error) {
	cmd := exec.Command("minikube", "start", "-p", config.Name)
	fmt.Println("creating minikube cluster...")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to start minikube: %s", err)
	}
	cmd = exec.Command("kubectl", "config", "current-context")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch id`: %s", err)
	}

	return &Cluster{Name: config.Name, Provider: "local"}, nil
}

// DeleteCluster implements CloudProvider.
func (l *LocalProvider) DeleteCluster(ctx context.Context, name string) error {
	panic("unimplemented")
}

// GetCluster implements CloudProvider.
func (l *LocalProvider) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	panic("unimplemented")
}

// GetProviderName implements CloudProvider.
func (l *LocalProvider) GetProviderName() string {
	panic("unimplemented")
}

// GetSupportedRegions implements CloudProvider.
func (l *LocalProvider) GetSupportedRegions() []string {
	panic("unimplemented")
}

// GetSupportedVersions implements CloudProvider.
func (l *LocalProvider) GetSupportedVersions() []string {
	panic("unimplemented")
}

// ScaleCluster implements CloudProvider.
func (l *LocalProvider) ScaleCluster(ctx context.Context, name string, nodeCount int) error {
	panic("unimplemented")
}

// UpdateCluster implements CloudProvider.
func (l *LocalProvider) UpdateCluster(ctx context.Context, name string, config *ClusterConfig) (*Cluster, error) {
	panic("unimplemented")
}

// ValidateConfig implements CloudProvider.
func (l *LocalProvider) ValidateConfig(config *ClusterConfig) error {
	panic("unimplemented")
}

var _ Provider = (*LocalProvider)(nil)
