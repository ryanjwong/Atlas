package state

import (
	"context"
	"time"
)

// StateManager defines the interface for state management
type StateManager interface {
	// Core state operations
	SaveClusterState(ctx context.Context, state *ClusterState) error
	GetClusterState(ctx context.Context, name string) (*ClusterState, error)
	ListClusters(ctx context.Context) ([]*ClusterState, error)
	DeleteClusterState(ctx context.Context, name string) error

	// Resource operations
	SaveResource(ctx context.Context, resource *Resource) error
	GetResource(ctx context.Context, clusterName, resourceID string) (*Resource, error)
	ListResources(ctx context.Context, clusterName string) ([]*Resource, error)
	DeleteResource(ctx context.Context, clusterName, resourceID string) error

	// Locking operations
	AcquireLock(ctx context.Context, resource string, timeout time.Duration) (Lock, error)
	ReleaseLock(ctx context.Context, lock Lock) error

	// Backup operations
	// CreateBackup(ctx context.Context) (*Backup, error)
	// RestoreBackup(ctx context.Context, backupID string) error
	// ListBackups(ctx context.Context) ([]*Backup, error)
	// DeleteBackup(ctx context.Context, backupID string) error

	// Maintenance operations
	Migrate(ctx context.Context, target StateManager) error
	Cleanup(ctx context.Context, olderThan time.Duration) error
	Validate(ctx context.Context) error

	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Health(ctx context.Context) error
}

// Lock represents a distributed lock
type Lock interface {
	ID() string
	Resource() string
	AcquiredAt() time.Time
	ExpiresAt() time.Time
	Refresh(ctx context.Context, timeout time.Duration) error
}

// ClusterState represents the state of a cluster
type ClusterState struct {
	Name      string            `json:"name"`
	Provider  string            `json:"provider"`
	Region    string            `json:"region"`
	Status    string            `json:"status"`
	NodeCount int               `json:"nodeCount"`
	Version   string            `json:"version"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	CreatedBy string            `json:"createdBy"`
	Metadata  map[string]string `json:"metadata"`
	Resources []*Resource       `json:"resources"`
	//Credentials *EncryptedData    `json:"credentials,omitempty"`
	LastBackup *time.Time `json:"lastBackup,omitempty"`
}

// Resource represents a cluster resource
type Resource struct {
	ID           string         `json:"id"`
	ClusterName  string         `json:"clusterName"`
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Namespace    string         `json:"namespace,omitempty"`
	Status       string         `json:"status"`
	Config       map[string]any `json:"config"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	Dependencies []string       `json:"dependencies,omitempty"`
}
