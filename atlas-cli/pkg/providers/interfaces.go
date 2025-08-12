package providers

import (
	"context"
	"time"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/logsource"
)

// Provider defines the interface for providers
type Provider interface {
	// Cluster lifecycle operations (these will be logged by the provider's log source)
	CreateCluster(ctx context.Context, config *ClusterConfig) (*Cluster, error)
	DeleteCluster(ctx context.Context, name string) error
	StartCluster(ctx context.Context, name string) error
	StopCluster(ctx context.Context, name string) error
	ScaleCluster(ctx context.Context, name string, nodeCount int) error

	// Read operations (these read directly from the provider)
	GetCluster(ctx context.Context, name string) (*Cluster, error)
	ListClusters(ctx context.Context) ([]*Cluster, error)

	// Provider metadata
	GetProviderName() string
	ValidateConfig(config *ClusterConfig) error
	GetSupportedRegions() []string
	GetSupportedVersions() []string

	// Log source for operation history and cluster information
	GetLogSource() logsource.LogSource
}

// ClusterConfig represents cluster configuration
type ClusterConfig struct {
	Name           string            `yaml:"name"`
	Region         string            `yaml:"region"`
	Version        string            `yaml:"version"`
	NodeCount      int               `yaml:"nodeCount"`
	InstanceType   string            `yaml:"instanceType"`
	NetworkConfig  *NetworkConfig    `yaml:"networkConfig,omitempty"`
	SecurityConfig *SecurityConfig   `yaml:"securityConfig,omitempty"`
	ResourceConfig *ResourceConfig   `yaml:"resourceConfig,omitempty"`
	Tags           map[string]string `yaml:"tags,omitempty"`
}

// NetworkConfig defines networking configuration for clusters
type NetworkConfig struct {
	PodCIDR       string              `yaml:"podCIDR,omitempty"`
	ServiceCIDR   string              `yaml:"serviceCIDR,omitempty"`
	ClusterDNS    string              `yaml:"clusterDNS,omitempty"`
	NetworkPlugin string              `yaml:"networkPlugin,omitempty"`
	DNSPolicy     string              `yaml:"dnsPolicy,omitempty"`
	ExtraPortMaps []PortMapping       `yaml:"extraPortMaps,omitempty"`
	APIServerPort int                 `yaml:"apiServerPort,omitempty"`
	Ingress       *IngressConfig      `yaml:"ingress,omitempty"`
	LoadBalancer  *LoadBalancerConfig `yaml:"loadBalancer,omitempty"`
}

// PortMapping defines port mapping for exposing services
type PortMapping struct {
	HostPort      int    `yaml:"hostPort"`
	ContainerPort int    `yaml:"containerPort"`
	Protocol      string `yaml:"protocol,omitempty"`
	NodePort      int    `yaml:"nodePort,omitempty"`
}

// IngressConfig defines ingress controller configuration
type IngressConfig struct {
	Enabled    bool              `yaml:"enabled"`
	Controller string            `yaml:"controller,omitempty"`
	Config     map[string]string `yaml:"config,omitempty"`
}

// LoadBalancerConfig defines load balancer configuration
type LoadBalancerConfig struct {
	Enabled bool              `yaml:"enabled"`
	Type    string            `yaml:"type,omitempty"`
	Config  map[string]string `yaml:"config,omitempty"`
}

// SecurityConfig defines security configuration for clusters
type SecurityConfig struct {
	RBAC               *RBACConfig          `yaml:"rbac,omitempty"`
	PodSecurityPolicy  *PodSecurityConfig   `yaml:"podSecurityPolicy,omitempty"`
	NetworkPolicy      *NetworkPolicyConfig `yaml:"networkPolicy,omitempty"`
	Encryption         *EncryptionConfig    `yaml:"encryption,omitempty"`
	AuditLogging       *AuditConfig         `yaml:"auditLogging,omitempty"`
	ImageSecurity      *ImageSecurityConfig `yaml:"imageSecurity,omitempty"`
	AuthenticationMode string               `yaml:"authenticationMode,omitempty"`
	ServiceMesh        *ServiceMeshConfig   `yaml:"serviceMesh,omitempty"`
}

// RBACConfig defines role-based access control settings
type RBACConfig struct {
	Enabled bool              `yaml:"enabled"`
	Rules   []RBACRule        `yaml:"rules,omitempty"`
	Config  map[string]string `yaml:"config,omitempty"`
}

// RBACRule defines individual RBAC rules
type RBACRule struct {
	Name      string   `yaml:"name"`
	Resources []string `yaml:"resources"`
	Verbs     []string `yaml:"verbs"`
	Namespace string   `yaml:"namespace,omitempty"`
}

// PodSecurityConfig defines pod security policy settings
type PodSecurityConfig struct {
	Enabled             bool     `yaml:"enabled"`
	AllowedCapabilities []string `yaml:"allowedCapabilities,omitempty"`
	ForbiddenSysctls    []string `yaml:"forbiddenSysctls,omitempty"`
	RunAsNonRoot        bool     `yaml:"runAsNonRoot,omitempty"`
	SELinuxOptions      string   `yaml:"seLinuxOptions,omitempty"`
}

// NetworkPolicyConfig defines network policy settings
type NetworkPolicyConfig struct {
	Enabled       bool                `yaml:"enabled"`
	DefaultPolicy string              `yaml:"defaultPolicy,omitempty"`
	Rules         []NetworkPolicyRule `yaml:"rules,omitempty"`
}

// NetworkPolicyRule defines individual network policy rules
type NetworkPolicyRule struct {
	Name      string   `yaml:"name"`
	Namespace string   `yaml:"namespace"`
	Ingress   []string `yaml:"ingress,omitempty"`
	Egress    []string `yaml:"egress,omitempty"`
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	AtRest      bool   `yaml:"atRest"`
	InTransit   bool   `yaml:"inTransit"`
	Algorithm   string `yaml:"algorithm,omitempty"`
	KeyRotation bool   `yaml:"keyRotation,omitempty"`
}

// AuditConfig defines audit logging settings
type AuditConfig struct {
	Enabled   bool              `yaml:"enabled"`
	LogLevel  string            `yaml:"logLevel,omitempty"`
	LogPath   string            `yaml:"logPath,omitempty"`
	Retention int               `yaml:"retention,omitempty"`
	Config    map[string]string `yaml:"config,omitempty"`
}

// ImageSecurityConfig defines container image security settings
type ImageSecurityConfig struct {
	ScanEnabled            bool     `yaml:"scanEnabled"`
	AllowedRegistries      []string `yaml:"allowedRegistries,omitempty"`
	SignatureVerification  bool     `yaml:"signatureVerification,omitempty"`
	VulnerabilityThreshold string   `yaml:"vulnerabilityThreshold,omitempty"`
}

// ServiceMeshConfig defines service mesh settings
type ServiceMeshConfig struct {
	Enabled  bool              `yaml:"enabled"`
	Provider string            `yaml:"provider,omitempty"`
	Config   map[string]string `yaml:"config,omitempty"`
	mTLS     bool              `yaml:"mTLS,omitempty"`
}

// ResourceConfig defines resource policies and constraints
type ResourceConfig struct {
	Limits      *ResourceLimits    `yaml:"limits,omitempty"`
	Requests    *ResourceRequests  `yaml:"requests,omitempty"`
	Quotas      *ResourceQuotas    `yaml:"quotas,omitempty"`
	AutoScaling *AutoScalingConfig `yaml:"autoScaling,omitempty"`
	Storage     *StorageConfig     `yaml:"storage,omitempty"`
	Monitoring  *MonitoringConfig  `yaml:"monitoring,omitempty"`
}

// ResourceLimits defines resource limit constraints
type ResourceLimits struct {
	CPU              string `yaml:"cpu,omitempty"`
	Memory           string `yaml:"memory,omitempty"`
	Storage          string `yaml:"storage,omitempty"`
	EphemeralStorage string `yaml:"ephemeralStorage,omitempty"`
	GPUs             int    `yaml:"gpus,omitempty"`
}

// ResourceRequests defines resource request constraints
type ResourceRequests struct {
	CPU              string `yaml:"cpu,omitempty"`
	Memory           string `yaml:"memory,omitempty"`
	Storage          string `yaml:"storage,omitempty"`
	EphemeralStorage string `yaml:"ephemeralStorage,omitempty"`
}

// ResourceQuotas defines namespace-level resource quotas
type ResourceQuotas struct {
	Namespaces   map[string]NamespaceQuota `yaml:"namespaces,omitempty"`
	DefaultQuota *NamespaceQuota           `yaml:"defaultQuota,omitempty"`
}

// NamespaceQuota defines quota limits for a namespace
type NamespaceQuota struct {
	CPU     string `yaml:"cpu,omitempty"`
	Memory  string `yaml:"memory,omitempty"`
	Storage string `yaml:"storage,omitempty"`
	Pods    int    `yaml:"pods,omitempty"`
	PVCs    int    `yaml:"pvcs,omitempty"`
}

// AutoScalingConfig defines auto-scaling policies
type AutoScalingConfig struct {
	Enabled                 bool   `yaml:"enabled"`
	MinNodes                int    `yaml:"minNodes,omitempty"`
	MaxNodes                int    `yaml:"maxNodes,omitempty"`
	TargetCPU               int    `yaml:"targetCPU,omitempty"`
	TargetMemory            int    `yaml:"targetMemory,omitempty"`
	ScaleUpDelay            string `yaml:"scaleUpDelay,omitempty"`
	ScaleDownDelay          string `yaml:"scaleDownDelay,omitempty"`
	HorizontalPodAutoScaler bool   `yaml:"horizontalPodAutoScaler,omitempty"`
}

// StorageConfig defines storage policies
type StorageConfig struct {
	DefaultStorageClass string               `yaml:"defaultStorageClass,omitempty"`
	StorageClasses      []StorageClassConfig `yaml:"storageClasses,omitempty"`
	VolumeExpansion     bool                 `yaml:"volumeExpansion,omitempty"`
	SnapshotController  bool                 `yaml:"snapshotController,omitempty"`
}

// StorageClassConfig defines storage class configuration
type StorageClassConfig struct {
	Name        string            `yaml:"name"`
	Provisioner string            `yaml:"provisioner"`
	Parameters  map[string]string `yaml:"parameters,omitempty"`
	Default     bool              `yaml:"default,omitempty"`
}

// MonitoringConfig defines monitoring and observability settings
type MonitoringConfig struct {
	Enabled        bool              `yaml:"enabled"`
	Prometheus     *PrometheusConfig `yaml:"prometheus,omitempty"`
	Grafana        *GrafanaConfig    `yaml:"grafana,omitempty"`
	LogAggregation *LogConfig        `yaml:"logAggregation,omitempty"`
	Tracing        *TracingConfig    `yaml:"tracing,omitempty"`
}

// PrometheusConfig defines Prometheus monitoring settings
type PrometheusConfig struct {
	Enabled        bool              `yaml:"enabled"`
	Retention      string            `yaml:"retention,omitempty"`
	StorageSize    string            `yaml:"storageSize,omitempty"`
	ScrapeInterval string            `yaml:"scrapeInterval,omitempty"`
	ExternalURL    string            `yaml:"externalURL,omitempty"`
	Config         map[string]string `yaml:"config,omitempty"`
}

// GrafanaConfig defines Grafana dashboard settings
type GrafanaConfig struct {
	Enabled     bool              `yaml:"enabled"`
	AdminUser   string            `yaml:"adminUser,omitempty"`
	Persistence bool              `yaml:"persistence,omitempty"`
	Config      map[string]string `yaml:"config,omitempty"`
}

// LogConfig defines log aggregation settings
type LogConfig struct {
	Enabled   bool              `yaml:"enabled"`
	Backend   string            `yaml:"backend,omitempty"`
	Retention string            `yaml:"retention,omitempty"`
	Config    map[string]string `yaml:"config,omitempty"`
}

// TracingConfig defines distributed tracing settings
type TracingConfig struct {
	Enabled    bool              `yaml:"enabled"`
	Backend    string            `yaml:"backend,omitempty"`
	SampleRate float64           `yaml:"sampleRate,omitempty"`
	Config     map[string]string `yaml:"config,omitempty"`
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
