# Atlas CLI Project Structure

## Directory Layout

```
atlas-cli/
├── cmd/                           # Cobra command implementations
│   ├── root.go                    # Root command and global flags
│   ├── cluster.go                 # Cluster management commands
│   ├── config.go                  # Configuration management commands
│   ├── git.go                     # Git operations commands
│   ├── monitoring.go              # Monitoring setup commands
│   ├── ansible.go                 # Ansible playbook commands
│   ├── template.go                # Template management commands
│   ├── workflow.go                # Workflow automation commands
│   ├── state.go                   # State management commands
│   └── backup.go                  # Backup and restore commands
├── pkg/                           # Reusable packages
│   ├── providers/                 # Cloud provider implementations
│   │   ├── interfaces.go          # Provider interfaces
│   │   ├── aws/                   # AWS provider
│   │   │   ├── eks.go             # EKS cluster management
│   │   │   ├── ec2.go             # EC2 resource management
│   │   │   └── iam.go             # IAM role management
│   │   ├── gcp/                   # Google Cloud provider
│   │   │   ├── gke.go             # GKE cluster management
│   │   │   ├── compute.go         # Compute Engine management
│   │   │   └── iam.go             # IAM management
│   │   └── azure/                 # Azure provider
│   │       ├── aks.go             # AKS cluster management
│   │       ├── vm.go              # Virtual Machine management
│   │       └── rbac.go            # Role-based access control
│   ├── state/                     # State management
│   │   ├── interfaces.go          # State manager interfaces
│   │   ├── local/                 # Local storage implementation
│   │   │   ├── sqlite.go          # SQLite implementation
│   │   │   └── migrations/        # Database migrations
│   │   │       ├── 001_init.sql
│   │   │       ├── 002_resources.sql
│   │   │       └── 003_audit.sql
│   │   ├── remote/                # Remote storage implementations
│   │   │   ├── etcd.go            # etcd implementation
│   │   │   ├── postgresql.go      # PostgreSQL implementation
│   │   │   └── consul.go          # Consul implementation
│   │   └── models.go              # State data models
│   ├── backup/                    # Backup management
│   │   ├── interfaces.go          # Backup interfaces
│   │   ├── manager.go             # Backup manager implementation
│   │   ├── storage/               # Backup storage providers
│   │   │   ├── local.go           # Local filesystem storage
│   │   │   ├── s3.go              # AWS S3 storage
│   │   │   ├── gcs.go             # Google Cloud Storage
│   │   │   └── azure.go           # Azure Blob Storage
│   │   └── compression.go         # Backup compression utilities
│   ├── encryption/                # Data encryption
│   │   ├── interfaces.go          # Encryption interfaces
│   │   ├── manager.go             # Encryption manager
│   │   ├── providers/             # Key providers
│   │   │   ├── vault.go           # HashiCorp Vault
│   │   │   ├── kms.go             # AWS KMS
│   │   │   └── local.go           # Local key storage
│   │   └── utils.go               # Encryption utilities
│   ├── cache/                     # Caching layer
│   │   ├── interfaces.go          # Cache interfaces
│   │   ├── memory.go              # In-memory cache
│   │   ├── redis.go               # Redis cache
│   │   └── manager.go             # Cache manager
│   ├── monitoring/                # Monitoring components
│   │   ├── prometheus/            # Prometheus integration
│   │   │   ├── installer.go       # Prometheus installation
│   │   │   ├── config.go          # Configuration management
│   │   │   └── rules.go           # Alert rules management
│   │   ├── grafana/               # Grafana integration
│   │   │   ├── installer.go       # Grafana installation
│   │   │   ├── datasources.go     # Datasource configuration
│   │   │   └── dashboards.go      # Dashboard management
│   │   └── exporters/             # Metric exporters
│   │       ├── node.go            # Node exporter
│   │       ├── kube_state.go      # Kube-state-metrics
│   │       └── custom.go          # Custom exporters
│   ├── ansible/                   # Ansible integration
│   │   ├── executor.go            # Playbook executor
│   │   ├── inventory.go           # Inventory management
│   │   └── templates/             # Playbook templates
│   │       ├── argocd.yml         # ArgoCD installation
│   │       ├── monitoring.yml     # Monitoring stack
│   │       └── security.yml       # Security hardening
│   ├── git/                       # Git operations
│   │   ├── interfaces.go          # Git interfaces
│   │   ├── github.go              # GitHub API integration
│   │   ├── gitlab.go              # GitLab API integration
│   │   └── operations.go          # Git operations
│   ├── template/                  # Template management
│   │   ├── engine.go              # Template engine
│   │   ├── loader.go              # Template loader
│   │   └── builtin/               # Built-in templates
│   │       ├── cluster/           # Cluster templates
│   │       ├── monitoring/        # Monitoring templates
│   │       └── application/       # Application templates
│   ├── workflow/                  # Workflow engine
│   │   ├── interfaces.go          # Workflow interfaces
│   │   ├── engine.go              # Workflow execution engine
│   │   ├── parser.go              # Workflow definition parser
│   │   └── steps/                 # Workflow steps
│   │       ├── cluster.go         # Cluster creation step
│   │       ├── ansible.go         # Ansible execution step
│   │       └── monitoring.go      # Monitoring setup step
│   ├── config/                    # Configuration management
│   │   ├── loader.go              # Configuration loader
│   │   ├── validator.go           # Configuration validator
│   │   └── models.go              # Configuration models
│   ├── auth/                      # Authentication and authorization
│   │   ├── interfaces.go          # Auth interfaces
│   │   ├── providers/             # Auth providers
│   │   │   ├── oauth.go           # OAuth provider
│   │   │   ├── ldap.go            # LDAP provider
│   │   │   └── local.go           # Local auth
│   │   └── rbac.go                # Role-based access control
│   ├── k8s/                       # Kubernetes utilities
│   │   ├── client.go              # Kubernetes client wrapper
│   │   ├── resources.go           # Resource management
│   │   └── helm.go                # Helm chart integration
│   ├── utils/                     # Utility functions
│   │   ├── logger.go              # Structured logging
│   │   ├── errors.go              # Error handling
│   │   ├── http.go                # HTTP utilities
│   │   └── validation.go          # Input validation
│   └── version/                   # Version information
│       └── version.go             # Version constants
├── internal/                      # Internal packages (not for external use)
│   ├── metrics/                   # Internal metrics
│   │   ├── collector.go           # Metrics collection
│   │   └── server.go              # Metrics server
│   ├── health/                    # Health checking
│   │   ├── checker.go             # Health check implementation
│   │   └── endpoints.go           # Health endpoints
│   └── signals/                   # Signal handling
│       └── handler.go             # Signal handler
├── configs/                       # Configuration files
│   ├── atlas.yaml                 # Default configuration
│   ├── prometheus/                # Prometheus configurations
│   │   ├── prometheus.yml         # Main configuration
│   │   └── rules/                 # Alert rules
│   │       ├── kubernetes.yml     # Kubernetes alerts
│   │       └── application.yml    # Application alerts
│   ├── grafana/                   # Grafana configurations
│   │   ├── grafana.ini            # Main configuration
│   │   ├── datasources/           # Datasource configurations
│   │   │   └── prometheus.yml     # Prometheus datasource
│   │   └── dashboards/            # Dashboard definitions
│   │       ├── kubernetes.json    # Kubernetes dashboard
│   │       └── application.json   # Application dashboard
│   └── ansible/                   # Ansible configurations
│       ├── ansible.cfg            # Ansible configuration
│       └── inventory/             # Inventory templates
├── templates/                     # Template files
│   ├── cluster/                   # Cluster templates
│   │   ├── aws-eks.yaml           # AWS EKS template
│   │   ├── gcp-gke.yaml           # GCP GKE template
│   │   └── azure-aks.yaml         # Azure AKS template
│   ├── monitoring/                # Monitoring templates
│   │   ├── prometheus-values.yaml # Prometheus Helm values
│   │   └── grafana-values.yaml    # Grafana Helm values
│   └── workflows/                 # Workflow templates
│       ├── full-stack.yaml        # Full stack deployment
│       └── monitoring-only.yaml   # Monitoring setup only
├── scripts/                       # Build and deployment scripts
│   ├── build.sh                   # Build script
│   ├── test.sh                    # Test script
│   ├── install.sh                 # Installation script
│   └── migrate.sh                 # State migration script
├── docs/                          # Documentation
│   ├── README.md                  # Main documentation
│   ├── CONTRIBUTING.md            # Contribution guidelines
│   ├── CHANGELOG.md               # Change log
│   └── examples/                  # Usage examples
│       ├── basic-cluster.md       # Basic cluster creation
│       └── full-deployment.md     # Full deployment example
├── test/                          # Test files
│   ├── unit/                      # Unit tests
│   ├── integration/               # Integration tests
│   └── e2e/                       # End-to-end tests
├── deployments/                   # Deployment configurations
│   ├── docker/                    # Docker configurations
│   │   ├── Dockerfile             # Main Dockerfile
│   │   └── docker-compose.yml     # Docker Compose
│   └── kubernetes/                # Kubernetes manifests
│       ├── deployment.yaml        # Deployment manifest
│       └── service.yaml           # Service manifest
├── .github/                       # GitHub configurations
│   ├── workflows/                 # GitHub Actions
│   │   ├── ci.yml                 # Continuous Integration
│   │   ├── release.yml            # Release workflow
│   │   └── security.yml           # Security scanning
│   └── ISSUE_TEMPLATE/            # Issue templates
├── main.go                        # Application entry point
├── go.mod                         # Go module definition
├── go.sum                         # Go module checksums
├── Makefile                       # Build automation
├── .gitignore                     # Git ignore rules
├── LICENSE                        # License file
└── README.md                      # Project README
```

## Key Interface Files

### pkg/providers/interfaces.go
```go
package providers

import (
    "context"
    "time"
)

// CloudProvider defines the interface for cloud providers
type CloudProvider interface {
    // Cluster operations
    CreateCluster(ctx context.Context, config *ClusterConfig) (*Cluster, error)
    GetCluster(ctx context.Context, name string) (*Cluster, error)
    ListClusters(ctx context.Context) ([]*Cluster, error)
    UpdateCluster(ctx context.Context, name string, config *ClusterConfig) (*Cluster, error)
    DeleteCluster(ctx context.Context, name string) error
    ScaleCluster(ctx context.Context, name string, nodeCount int) error
    
    // Provider-specific operations
    GetProviderName() string
    ValidateConfig(config *ClusterConfig) error
    GetSupportedRegions() []string
    GetSupportedVersions() []string
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
    Tags           map[string]string `yaml:"tags,omitempty"`
}

// Cluster represents a cluster instance
type Cluster struct {
    ID             string            `json:"id"`
    Name           string            `json:"name"`
    Provider       string            `json:"provider"`
    Region         string            `json:"region"`
    Version        string            `json:"version"`
    Status         ClusterStatus     `json:"status"`
    NodeCount      int               `json:"nodeCount"`
    Endpoint       string            `json:"endpoint"`
    CreatedAt      time.Time         `json:"createdAt"`
    UpdatedAt      time.Time         `json:"updatedAt"`
    Tags           map[string]string `json:"tags"`
    KubeConfig     string            `json:"kubeConfig,omitempty"`
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
```

### pkg/state/interfaces.go
```go
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
    CreateBackup(ctx context.Context) (*Backup, error)
    RestoreBackup(ctx context.Context, backupID string) error
    ListBackups(ctx context.Context) ([]*Backup, error)
    DeleteBackup(ctx context.Context, backupID string) error
    
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
    Name         string            `json:"name"`
    Provider     string            `json:"provider"`
    Region       string            `json:"region"`
    Status       string            `json:"status"`
    NodeCount    int               `json:"nodeCount"`
    Version      string            `json:"version"`
    CreatedAt    time.Time         `json:"createdAt"`
    UpdatedAt    time.Time         `json:"updatedAt"`
    CreatedBy    string            `json:"createdBy"`
    Metadata     map[string]string `json:"metadata"`
    Resources    []*Resource       `json:"resources"`
    Credentials  *EncryptedData    `json:"credentials,omitempty"`
    LastBackup   *time.Time        `json:"lastBackup,omitempty"`
}

// Resource represents a cluster resource
type Resource struct {
    ID           string            `json:"id"`
    ClusterName  string            `json:"clusterName"`
    Type         string            `json:"type"`
    Name         string            `json:"name"`
    Namespace    string            `json:"namespace,omitempty"`
    Status       string            `json:"status"`
    Config       map[string]any    `json:"config"`
    CreatedAt    time.Time         `json:"createdAt"`
    UpdatedAt    time.Time         `json:"updatedAt"`
    Dependencies []string          `json:"dependencies,omitempty"`
}
```

### pkg/backup/interfaces.go
```go
package backup

import (
    "context"
    "io"
    "time"
)

// BackupManager defines the interface for backup management
type BackupManager interface {
    // Backup operations
    CreateBackup(ctx context.Context, config *BackupConfig) (*Backup, error)
    RestoreBackup(ctx context.Context, backupID string, config *RestoreConfig) error
    ListBackups(ctx context.Context, filter *BackupFilter) ([]*Backup, error)
    DeleteBackup(ctx context.Context, backupID string) error
    
    // Backup validation
    ValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error)
    
    // Backup scheduling
    ScheduleBackup(ctx context.Context, schedule *BackupSchedule) error
    UnscheduleBackup(ctx context.Context, scheduleID string) error
    ListSchedules(ctx context.Context) ([]*BackupSchedule, error)
}

// StorageProvider defines the interface for backup storage
type StorageProvider interface {
    // Storage operations
    Store(ctx context.Context, key string, data io.Reader) error
    Retrieve(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) ([]string, error)
    
    // Metadata operations
    GetMetadata(ctx context.Context, key string) (*StorageMetadata, error)
    SetMetadata(ctx context.Context, key string, metadata *StorageMetadata) error
    
    // Provider information
    GetProviderName() string
    GetCapabilities() []string
}

// BackupConfig represents backup configuration
type BackupConfig struct {
    Type         BackupType        `yaml:"type"`
    ClusterName  string            `yaml:"clusterName"`
    Components   []string          `yaml:"components"`
    Storage      *StorageConfig    `yaml:"storage"`
    Compression  bool              `yaml:"compression"`
    Encryption   bool              `yaml:"encryption"`
    Tags         map[string]string `yaml:"tags"`
}

// Backup represents a backup instance
type Backup struct {
    ID          string            `json:"id"`
    Type        BackupType        `json:"type"`
    ClusterName string            `json:"clusterName"`
    Components  []string          `json:"components"`
    CreatedAt   time.Time         `json:"createdAt"`
    Size        int64             `json:"size"`
    Checksum    string            `json:"checksum"`
    Status      BackupStatus      `json:"status"`
    Location    string            `json:"location"`
    ParentID    string            `json:"parentId,omitempty"`
    Tags        map[string]string `json:"tags"`
    Metadata    map[string]any    `json:"metadata"`
}

// BackupType represents the type of backup
type BackupType string

const (
    BackupTypeFull        BackupType = "full"
    BackupTypeIncremental BackupType = "incremental"
    BackupTypeDifferential BackupType = "differential"
)

// BackupStatus represents backup status
type BackupStatus string

const (
    BackupStatusPending   BackupStatus = "pending"
    BackupStatusRunning   BackupStatus = "running"
    BackupStatusCompleted BackupStatus = "completed"
    BackupStatusFailed    BackupStatus = "failed"
)
```

### pkg/monitoring/interfaces.go
```go
package monitoring

import (
    "context"
    "time"
)

// MonitoringManager defines the interface for monitoring management
type MonitoringManager interface {
    // Stack operations
    InstallStack(ctx context.Context, config *StackConfig) error
    UninstallStack(ctx context.Context, clusterName string) error
    GetStackStatus(ctx context.Context, clusterName string) (*StackStatus, error)
    
    // Component operations
    InstallComponent(ctx context.Context, component ComponentType, config *ComponentConfig) error
    UninstallComponent(ctx context.Context, component ComponentType, clusterName string) error
    GetComponentStatus(ctx context.Context, component ComponentType, clusterName string) (*ComponentStatus, error)
    
    // Dashboard operations
    ImportDashboard(ctx context.Context, dashboardID string, config *DashboardConfig) error
    ExportDashboard(ctx context.Context, dashboardID string) (*Dashboard, error)
    ListDashboards(ctx context.Context, clusterName string) ([]*Dashboard, error)
    DeleteDashboard(ctx context.Context, dashboardID string) error
    
    // Alert operations
    CreateAlertRule(ctx context.Context, rule *AlertRule) error
    UpdateAlertRule(ctx context.Context, ruleID string, rule *AlertRule) error
    DeleteAlertRule(ctx context.Context, ruleID string) error
    ListAlertRules(ctx context.Context, clusterName string) ([]*AlertRule, error)
}

// ComponentType represents monitoring component types
type ComponentType string

const (
    ComponentPrometheus   ComponentType = "prometheus"
    ComponentGrafana      ComponentType = "grafana"
    ComponentAlertManager ComponentType = "alertmanager"
    ComponentNodeExporter ComponentType = "node-exporter"
    ComponentKubeState    ComponentType = "kube-state-metrics"
)

// StackConfig represents monitoring stack configuration
type StackConfig struct {
    ClusterName string                       `yaml:"clusterName"`
    Namespace   string                       `yaml:"namespace"`
    Components  map[ComponentType]*ComponentConfig `yaml:"components"`
    Ingress     *IngressConfig               `yaml:"ingress,omitempty"`
    Storage     *StorageConfig               `yaml:"storage,omitempty"`
    Security    *SecurityConfig              `yaml:"security,omitempty"`
}

// ComponentConfig represents component configuration
type ComponentConfig struct {
    Enabled    bool              `yaml:"enabled"`
    Version    string            `yaml:"version,omitempty"`
    Resources  *ResourceRequirements `yaml:"resources,omitempty"`
    Values     map[string]any    `yaml:"values,omitempty"`
    Storage    *StorageConfig    `yaml:"storage,omitempty"`
}

// Dashboard represents a Grafana dashboard
type Dashboard struct {
    ID          string            `json:"id"`
    UID         string            `json:"uid"`
    Title       string            `json:"title"`
    Description string            `json:"description"`
    Tags        []string          `json:"tags"`
    CreatedAt   time.Time         `json:"createdAt"`
    UpdatedAt   time.Time         `json:"updatedAt"`
    Version     int               `json:"version"`
    Config      map[string]any    `json:"config"`
}

// AlertRule represents a Prometheus alert rule
type AlertRule struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Group       string            `json:"group"`
    Expression  string            `json:"expression"`
    Duration    time.Duration     `json:"duration"`
    Severity    string            `json:"severity"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    Enabled     bool              `json:"enabled"`
}
```

This project structure provides:

1. **Clear separation of concerns** with dedicated packages for each major functionality
2. **Comprehensive interfaces** that define contracts for all major components
3. **Modular design** that allows for easy testing and extension
4. **Configuration management** with proper templating support
5. **Complete state management** with multiple backend options
6. **Robust backup system** with multiple storage providers
7. **Monitoring integration** with Prometheus and Grafana
8. **Security considerations** with encryption and authentication
9. **Workflow automation** with a flexible execution engine
10. **Proper Go project structure** following best practices

Each interface is designed to be mockable for testing and extensible for future enhancements.