package providers

import (
	"context"
	"time"
)

// Provider defines the interface for providers
type Provider interface {
	// Cluster operations
	CreateCluster(ctx context.Context, config *ClusterConfig) (*Cluster, error)
	GetCluster(ctx context.Context, name string) (*Cluster, error)
	ListClusters(ctx context.Context) ([]*Cluster, error)
	UpdateCluster(ctx context.Context, name string, config *ClusterConfig) (*Cluster, error)
	DeleteCluster(ctx context.Context, name string) error
	ScaleCluster(ctx context.Context, name string, nodeCount int) error
	StartCluster(ctx context.Context, name string) error
	StopCluster(ctx context.Context, name string) error

	// Provider-specific operations
	GetProviderName() string
	ValidateConfig(config *ClusterConfig) error
	GetSupportedRegions() []string
	GetSupportedVersions() []string
}

// ClusterConfig represents cluster configuration
type ClusterConfig struct {
	Name         string `yaml:"name"`
	Region       string `yaml:"region"`
	Version      string `yaml:"version"`
	NodeCount    int    `yaml:"nodeCount"`
	InstanceType string `yaml:"instanceType"`
	// NetworkConfig  *NetworkConfig    `yaml:"networkConfig,omitempty"`
	// SecurityConfig *SecurityConfig   `yaml:"securityConfig,omitempty"`
	Tags map[string]string `yaml:"tags,omitempty"`
}

// Cluster represents a cluster instance
type Cluster struct {
	Name       string            `json:"name"`
	Provider   string            `json:"provider"`
	Region     string            `json:"region"`
	Version    string            `json:"version"`
	Status     ClusterStatus     `json:"status"`
	NodeCount  int               `json:"nodeCount"`
	Endpoint   string            `json:"endpoint"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
	Tags       map[string]string `json:"tags"`
	KubeConfig string            `json:"kubeConfig,omitempty"`
}

// ClusterStatus represents cluster status
type ClusterStatus string

const (
	ClusterStatusPending  ClusterStatus = "pending"
	ClusterStatusRunning  ClusterStatus = "running"
	ClusterStatusStopped  ClusterStatus = "stopped"
	ClusterStatusError    ClusterStatus = "error"
	ClusterStatusDeleting ClusterStatus = "deleting"
)
