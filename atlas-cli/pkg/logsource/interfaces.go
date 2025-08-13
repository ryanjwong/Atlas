package logsource

import (
	"context"
	"time"
)

// LogSource defines the interface for reading cluster operations and status from log sources
type LogSource interface {
	// Get operation history for a specific cluster
	GetClusterHistory(ctx context.Context, clusterName string, limit int) ([]*OperationHistory, error)
	
	// Get operation history for all clusters  
	GetAllClustersHistory(ctx context.Context, limit int) (map[string][]*OperationHistory, error)
	
	// Get the source name for identification
	GetSourceName() string
}

// OperationHistory represents a cluster operation from logs
type OperationHistory struct {
	ID               int                    `json:"id"`
	ClusterName      string                 `json:"cluster_name"`
	OperationType    OperationType          `json:"operation_type"`
	OperationStatus  OperationStatus        `json:"operation_status"`
	StartedAt        time.Time              `json:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	DurationMS       *float64               `json:"duration_ms,omitempty"`
	UserID           string                 `json:"user_id"`
	OperationDetails map[string]interface{} `json:"operation_details,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	Metadata         map[string]string      `json:"metadata,omitempty"`
}

// ClusterInfo represents basic cluster information from logs
type ClusterInfo struct {
	Name         string            `json:"name"`
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	Status       StatusType        `json:"status"`
	NodeCount    int               `json:"nodeCount"`
	Version      string            `json:"version"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	Tags         map[string]string `json:"tags"`
	LastActivity *time.Time        `json:"lastActivity,omitempty"`
}

// ClusterStatus represents the current status of a cluster
type ClusterStatus struct {
	Name      string            `json:"name"`
	Status    StatusType        `json:"status"`
	NodeCount int               `json:"nodeCount"`
	Version   string            `json:"version"`
	Endpoint  string            `json:"endpoint"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Operation types that can be logged
type OperationType string

const (
	OpTypeCreate OperationType = "create"
	OpTypeStart  OperationType = "start"
	OpTypeStop   OperationType = "stop"
	OpTypeDelete OperationType = "delete"
	OpTypeScale  OperationType = "scale"
	OpTypeUpdate OperationType = "update"
)

// Operation status from logs
type OperationStatus string

const (
	OpStatusStarted   OperationStatus = "started"
	OpStatusRunning   OperationStatus = "running"
	OpStatusCompleted OperationStatus = "completed"
	OpStatusFailed    OperationStatus = "failed"
	OpStatusCanceled  OperationStatus = "canceled"
)

// Cluster status types
type StatusType string

const (
	StatusPending  StatusType = "pending"
	StatusRunning  StatusType = "running"
	StatusStopped  StatusType = "stopped"
	StatusError    StatusType = "error"
	StatusDeleting StatusType = "deleting"
)