package monitoring

import (
	"context"
	"time"
)

type Monitor interface {
	CheckClusterHealth(ctx context.Context, clusterName string) (*HealthStatus, error)
	GetClusterMetrics(ctx context.Context, clusterName string) (*ClusterMetrics, error)
	StartMonitoring(ctx context.Context, config *MonitoringConfig) error
	StopMonitoring(ctx context.Context, clusterName string) error
	GetMonitorName() string
}

type HealthStatus struct {
	ClusterName      string               `json:"cluster_name"`
	OverallStatus    ClusterHealthStatus  `json:"overall_status"`
	ControlPlane     *ControlPlaneHealth  `json:"control_plane"`
	Nodes            []NodeHealth         `json:"nodes"`
	Pods             *PodHealth           `json:"pods"`
	Services         *ServiceHealth       `json:"services"`
	LastChecked      time.Time            `json:"last_checked"`
	CheckDuration    time.Duration        `json:"check_duration"`
	Warnings         []string             `json:"warnings,omitempty"`
	Errors           []string             `json:"errors,omitempty"`
}

type ClusterMetrics struct {
	ClusterName     string              `json:"cluster_name"`
	Timestamp       time.Time           `json:"timestamp"`
	NodeMetrics     []NodeMetrics       `json:"node_metrics"`
	PodMetrics      []PodMetrics        `json:"pod_metrics"`
	ResourceUsage   *ResourceUsage      `json:"resource_usage"`
	NetworkMetrics  *NetworkMetrics     `json:"network_metrics,omitempty"`
	StorageMetrics  *StorageMetrics     `json:"storage_metrics,omitempty"`
}

type MonitoringConfig struct {
	ClusterNames     []string      `json:"cluster_names"`
	CheckInterval    time.Duration `json:"check_interval"`
	MetricsInterval  time.Duration `json:"metrics_interval"`
	AlertThresholds  *AlertThresholds `json:"alert_thresholds,omitempty"`
	EnableAlerts     bool          `json:"enable_alerts"`
	LogPath          string        `json:"log_path,omitempty"`
}

type ClusterHealthStatus string

const (
	HealthStatusHealthy    ClusterHealthStatus = "healthy"
	HealthStatusWarning    ClusterHealthStatus = "warning"
	HealthStatusUnhealthy  ClusterHealthStatus = "unhealthy"
	HealthStatusUnknown    ClusterHealthStatus = "unknown"
)

type ControlPlaneHealth struct {
	APIServer          ComponentStatus `json:"api_server"`
	Scheduler          ComponentStatus `json:"scheduler"`
	ControllerManager  ComponentStatus `json:"controller_manager"`
	Etcd               ComponentStatus `json:"etcd"`
}

type ComponentStatus struct {
	Status    ComponentHealthStatus `json:"status"`
	Message   string                `json:"message,omitempty"`
	LastCheck time.Time             `json:"last_check"`
}

type ComponentHealthStatus string

const (
	ComponentHealthy   ComponentHealthStatus = "healthy"
	ComponentUnhealthy ComponentHealthStatus = "unhealthy" 
	ComponentUnknown   ComponentHealthStatus = "unknown"
)

type NodeHealth struct {
	Name        string            `json:"name"`
	Status      NodeHealthStatus  `json:"status"`
	Ready       bool              `json:"ready"`
	Conditions  []NodeCondition   `json:"conditions"`
	Resources   *NodeResources    `json:"resources,omitempty"`
	Version     string            `json:"version"`
	LastChecked time.Time         `json:"last_checked"`
}

type NodeHealthStatus string

const (
	NodeHealthy     NodeHealthStatus = "healthy"
	NodeNotReady    NodeHealthStatus = "not_ready"
	NodeUnknown     NodeHealthStatus = "unknown"
)

type NodeCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"last_transition_time"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}

type NodeResources struct {
	CPUCapacity      string `json:"cpu_capacity"`
	MemoryCapacity   string `json:"memory_capacity"`
	StorageCapacity  string `json:"storage_capacity,omitempty"`
	CPUAllocatable   string `json:"cpu_allocatable"`
	MemoryAllocatable string `json:"memory_allocatable"`
}

type PodHealth struct {
	TotalPods     int                     `json:"total_pods"`
	RunningPods   int                     `json:"running_pods"`
	PendingPods   int                     `json:"pending_pods"`
	FailedPods    int                     `json:"failed_pods"`
	SucceededPods int                     `json:"succeeded_pods"`
	UnknownPods   int                     `json:"unknown_pods"`
	PodsByPhase   map[string]int          `json:"pods_by_phase"`
	CriticalPods  []CriticalPodInfo       `json:"critical_pods,omitempty"`
	Namespaces    map[string]*NamespaceHealth `json:"namespaces"`
}

type CriticalPodInfo struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Phase      string `json:"phase"`
	Message    string `json:"message,omitempty"`
	RestartCount int  `json:"restart_count"`
}

type NamespaceHealth struct {
	Name        string `json:"name"`
	TotalPods   int    `json:"total_pods"`
	HealthyPods int    `json:"healthy_pods"`
	Status      string `json:"status"`
}

type ServiceHealth struct {
	TotalServices     int            `json:"total_services"`
	HealthyServices   int            `json:"healthy_services"`
	UnhealthyServices int            `json:"unhealthy_services"`
	ServicesByType    map[string]int `json:"services_by_type"`
}

type NodeMetrics struct {
	NodeName      string        `json:"node_name"`
	CPUUsage      ResourceValue `json:"cpu_usage"`
	MemoryUsage   ResourceValue `json:"memory_usage"`
	StorageUsage  ResourceValue `json:"storage_usage,omitempty"`
	NetworkRx     ResourceValue `json:"network_rx,omitempty"`
	NetworkTx     ResourceValue `json:"network_tx,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
}

type PodMetrics struct {
	PodName       string                    `json:"pod_name"`
	Namespace     string                    `json:"namespace"`
	CPUUsage      ResourceValue             `json:"cpu_usage"`
	MemoryUsage   ResourceValue             `json:"memory_usage"`
	Containers    map[string]ContainerMetrics `json:"containers"`
	Timestamp     time.Time                 `json:"timestamp"`
}

type ContainerMetrics struct {
	CPUUsage    ResourceValue `json:"cpu_usage"`
	MemoryUsage ResourceValue `json:"memory_usage"`
}

type ResourceValue struct {
	Value string  `json:"value"`
	Usage float64 `json:"usage_percent,omitempty"`
}

type ResourceUsage struct {
	TotalCPU      ResourceValue `json:"total_cpu"`
	TotalMemory   ResourceValue `json:"total_memory"`
	TotalStorage  ResourceValue `json:"total_storage,omitempty"`
	UsedCPU       ResourceValue `json:"used_cpu"`
	UsedMemory    ResourceValue `json:"used_memory"`
	UsedStorage   ResourceValue `json:"used_storage,omitempty"`
	CPUPercentage    float64   `json:"cpu_percentage"`
	MemoryPercentage float64   `json:"memory_percentage"`
	StoragePercentage float64  `json:"storage_percentage,omitempty"`
}

type NetworkMetrics struct {
	TotalRx    ResourceValue `json:"total_rx"`
	TotalTx    ResourceValue `json:"total_tx"`
	PacketsRx  int64         `json:"packets_rx,omitempty"`
	PacketsTx  int64         `json:"packets_tx,omitempty"`
	ErrorsRx   int64         `json:"errors_rx,omitempty"`
	ErrorsTx   int64         `json:"errors_tx,omitempty"`
}

type StorageMetrics struct {
	TotalCapacity     ResourceValue              `json:"total_capacity"`
	UsedCapacity      ResourceValue              `json:"used_capacity"`
	AvailableCapacity ResourceValue              `json:"available_capacity"`
	UsagePercentage   float64                    `json:"usage_percentage"`
	VolumesByType     map[string]int             `json:"volumes_by_type"`
	VolumeMetrics     []PersistentVolumeMetrics  `json:"volume_metrics,omitempty"`
}

type PersistentVolumeMetrics struct {
	Name              string        `json:"name"`
	Capacity          ResourceValue `json:"capacity"`
	Used              ResourceValue `json:"used"`
	Available         ResourceValue `json:"available"`
	UsagePercentage   float64       `json:"usage_percentage"`
	StorageClass      string        `json:"storage_class"`
}

type AlertThresholds struct {
	CPUWarning       float64 `json:"cpu_warning"`
	CPUCritical      float64 `json:"cpu_critical"`
	MemoryWarning    float64 `json:"memory_warning"`
	MemoryCritical   float64 `json:"memory_critical"`
	StorageWarning   float64 `json:"storage_warning"`
	StorageCritical  float64 `json:"storage_critical"`
	NodeDownCount    int     `json:"node_down_count"`
	PodFailureRate   float64 `json:"pod_failure_rate"`
}

type MonitoringEvent struct {
	ID            string              `json:"id"`
	ClusterName   string              `json:"cluster_name"`
	EventType     MonitoringEventType `json:"event_type"`
	Severity      EventSeverity       `json:"severity"`
	Message       string              `json:"message"`
	Details       map[string]interface{} `json:"details,omitempty"`
	Timestamp     time.Time           `json:"timestamp"`
	Resolved      bool                `json:"resolved"`
	ResolvedAt    *time.Time          `json:"resolved_at,omitempty"`
}

type MonitoringEventType string

const (
	EventTypeHealthCheck   MonitoringEventType = "health_check"
	EventTypeMetrics       MonitoringEventType = "metrics"
	EventTypeAlert         MonitoringEventType = "alert"
	EventTypeStatusChange  MonitoringEventType = "status_change"
)

type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityCritical EventSeverity = "critical"
)