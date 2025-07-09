# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
# Build the CLI binary
go build -o atlas-cli

# Run the CLI locally
go run main.go [command]

# Run specific commands
go run main.go cluster list
go run main.go cluster create test-cluster --provider local --nodes 1
```

### Testing and Quality
```bash
# Run tests (when implemented)
go test ./...

# Run tests with verbose output
go test -v ./...

# Clean up dependencies
go mod tidy

# Check for unused dependencies
go mod why
```

### Database Operations
```bash
# The SQLite database is stored at ./state.db
# You can examine it using sqlite3 if needed
sqlite3 state.db ".tables"
sqlite3 state.db ".schema clusters"
```

## Architecture Overview

Atlas CLI is a Kubernetes cluster management tool built with Go and Cobra CLI framework. The architecture follows an interface-driven design with clear separation of concerns.

### Core Components

1. **CLI Framework**: Built on Cobra for command-line interface
2. **Provider System**: Pluggable architecture for different cloud providers
3. **State Management**: SQLite-based persistence with interfaces for other backends
4. **Service Layer**: Internal services for cross-cutting functionality

### Key Architectural Patterns

- **Interface-Driven Design**: All major components implement interfaces (Provider, StateManager)
- **Plugin Architecture**: Providers can be added without modifying core code
- **Command Pattern**: Each CLI command is a separate Cobra command
- **Repository Pattern**: State management abstracted behind interfaces

## Code Organization

```
atlas-cli/
├── cmd/                    # Cobra command implementations
│   ├── root.go            # Root command with global flags
│   ├── cluster.go         # Cluster management commands
│   ├── config.go          # Configuration commands
│   └── status.go          # Status reporting commands
├── internal/services/      # Internal service layer
│   └── services.go        # Service container and initialization
├── pkg/providers/          # Provider implementations
│   ├── interfaces.go      # Provider interface definitions
│   └── local.go           # Local/minikube provider
├── pkg/state/             # State management
│   ├── interfaces.go      # State management interfaces
│   └── sqlite.go          # SQLite implementation
├── main.go                # Application entry point
└── state.db               # SQLite database file
```

## Provider System

### Adding a New Provider

1. Implement the `Provider` interface in `pkg/providers/interfaces.go`:
```go
type Provider interface {
    CreateCluster(ctx context.Context, config *ClusterConfig) (*Cluster, error)
    GetCluster(ctx context.Context, name string) (*Cluster, error)
    UpdateCluster(ctx context.Context, name string, config *ClusterConfig) (*Cluster, error)
    DeleteCluster(ctx context.Context, name string) error
    ScaleCluster(ctx context.Context, name string, nodeCount int) error
    StartCluster(ctx context.Context, name string) error
    StopCluster(ctx context.Context, name string) error
    GetProviderName() string
    ValidateConfig(config *ClusterConfig) error
    GetSupportedRegions() []string
    GetSupportedVersions() []string
}
```

2. Create a new file in `pkg/providers/` (e.g., `aws.go`, `gcp.go`)
3. Register the provider in the command initialization

### Local Provider Implementation

The local provider (`pkg/providers/local.go`) implements minikube cluster management:
- Uses `minikube` CLI commands for cluster operations
- Supports multi-node clusters with `--nodes` flag
- Properly detects node count and Kubernetes version
- Handles cluster lifecycle (create, start, stop, delete, scale)

## State Management

### SQLite Backend

The current implementation uses SQLite for state persistence:
- Database file: `./state.db`
- Schema defined in `pkg/state/sqlite.go`
- Tables: `clusters`, `cluster_resources`, `state_locks`

### Adding New State Backends

1. Implement the `StateManager` interface in `pkg/state/interfaces.go`
2. Create new implementation file (e.g., `etcd.go`, `postgresql.go`)
3. Update service initialization in `internal/services/services.go`

## Command Structure

### Adding New Commands

1. Create new command in `cmd/` directory
2. Define Cobra command structure with proper flags
3. Implement command logic using the service layer
4. Register command in `init()` function

### Command Patterns

- Use `GetServices()` to access the service container
- Handle both text and JSON output formats
- Implement proper error handling with descriptive messages
- Use context for all operations

## Development Guidelines

### Code Style
- Follow Go coding conventions
- Use interfaces for testability
- Implement proper error handling
- Support both verbose and quiet modes

### Database Operations
- Always use prepared statements for SQL operations
- Handle database migrations properly
- Use transactions for complex operations
- Implement proper connection management

### Provider Development
- Follow the established provider interface
- Implement proper validation in `ValidateConfig()`
- Handle cloud provider rate limiting
- Support both synchronous and asynchronous operations

### Testing Strategy
- Unit tests for individual components
- Integration tests for provider interactions
- End-to-end tests for complete workflows
- Mock external dependencies for testing

## Future Architecture

See `PROJECT_STRUCTURE.md` for detailed plans on:
- Multi-cloud provider support (AWS, GCP, Azure)
- Advanced state management (etcd, PostgreSQL)
- Monitoring integration (Prometheus, Grafana)
- GitOps workflows (ArgoCD integration)
- Backup and disaster recovery
- Security and compliance features

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/mattn/go-sqlite3` - SQLite driver
- Standard Go libraries for HTTP, JSON, and system operations

## Important Notes

- The codebase follows a strict no-comments policy
- All provider implementations should handle errors gracefully
- State management operations should be atomic where possible
- CLI commands should support both interactive and scripted usage
- Configuration should be environment-aware (dev, staging, production)