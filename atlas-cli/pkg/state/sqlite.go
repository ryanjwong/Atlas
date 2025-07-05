package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const sqliteStateSchema string = `
CREATE TABLE IF NOT EXISTS clusters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    provider TEXT NOT NULL,
    region TEXT NOT NULL,
    node_count INTEGER NOT NULL DEFAULT 3,
    version TEXT NOT NULL,
    config TEXT NOT NULL,
    metadata TEXT,
    credentials BLOB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL DEFAULT 'atlas-cli'
);

CREATE TABLE IF NOT EXISTS cluster_resources (
    id TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_name TEXT NOT NULL,
    namespace TEXT,
    status TEXT NOT NULL,
    config TEXT NOT NULL,
    dependencies TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE,
    UNIQUE(cluster_id, resource_type, resource_name, namespace)
);

CREATE TABLE IF NOT EXISTS state_locks (
    resource_name TEXT PRIMARY KEY,
    lock_id TEXT NOT NULL,
    acquired_by TEXT NOT NULL,
    acquired_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    metadata TEXT
);

CREATE INDEX IF NOT EXISTS idx_clusters_name ON clusters(name);
CREATE INDEX IF NOT EXISTS idx_clusters_provider ON clusters(provider);
CREATE INDEX IF NOT EXISTS idx_resources_cluster_id ON cluster_resources(cluster_id);
CREATE INDEX IF NOT EXISTS idx_locks_expires_at ON state_locks(expires_at);
`

type SQLiteStateManager struct {
	db *sql.DB
}

func NewSQLiteStateManager(path string) (*SQLiteStateManager, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	if _, err = db.Exec(sqliteStateSchema); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStateManager{
		db: db,
	}, nil
}

// AcquireLock implements StateManager.
func (s *SQLiteStateManager) AcquireLock(ctx context.Context, resource string, timeout time.Duration) (Lock, error) {
	panic("unimplemented")
}

// Cleanup implements StateManager.
func (s *SQLiteStateManager) Cleanup(ctx context.Context, olderThan time.Duration) error {
	panic("unimplemented")
}

// Connect implements StateManager.
func (s *SQLiteStateManager) Connect(ctx context.Context) error {
	panic("unimplemented")
}

// DeleteClusterState implements StateManager.
func (s *SQLiteStateManager) DeleteClusterState(ctx context.Context, name string) error {
	panic("unimplemented")
}

// DeleteResource implements StateManager.
func (s *SQLiteStateManager) DeleteResource(ctx context.Context, clusterName string, resourceID string) error {
	panic("unimplemented")
}

// Disconnect implements StateManager.
func (s *SQLiteStateManager) Disconnect(ctx context.Context) error {
	panic("unimplemented")
}

// GetClusterState implements StateManager.
func (s *SQLiteStateManager) GetClusterState(ctx context.Context, name string) (*ClusterState, error) {
	panic("unimplemented")
}

// GetResource implements StateManager.
func (s *SQLiteStateManager) GetResource(ctx context.Context, clusterName string, resourceID string) (*Resource, error) {
	panic("unimplemented")
}

// Health implements StateManager.
func (s *SQLiteStateManager) Health(ctx context.Context) error {
	panic("unimplemented")
}

// ListClusters implements StateManager.
func (s *SQLiteStateManager) ListClusters(ctx context.Context) ([]*ClusterState, error) {
	query := `
		SELECT 
			id, name, provider, region, node_count, version,
			config, metadata, created_at, updated_at, created_by
		FROM clusters 
		ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []*ClusterState

	for rows.Next() {
		var cluster ClusterState
		var configJSON, metadataJSON string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&cluster.ID,
			&cluster.Name,
			&cluster.Provider,
			&cluster.Region,
			&cluster.NodeCount,
			&cluster.Version,
			&configJSON,
			&metadataJSON,
			&createdAt,
			&updatedAt,
			&cluster.CreatedBy,
		)

		if err != nil {
			return nil, err
		}

		// Parse JSON fields
		json.Unmarshal([]byte(configJSON), &cluster.Config)
		json.Unmarshal([]byte(metadataJSON), &cluster.Metadata)

		cluster.CreatedAt = createdAt
		cluster.UpdatedAt = updatedAt

		clusters = append(clusters, &cluster)
	}

	return clusters, rows.Err()
}

// CreateCluster implements StateManager.
func (s *SQLiteStateManager) CreateCluster(ctx context.Context, name string, provider string, region string, nodes int) error {
	query := `INSERT INTO clusters (name, provider, region, node_count, version,
			config, metadata) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, name, provider, region, nodes, "1", "test", "test")
	return err
}

// ListResources implements StateManager.
func (s *SQLiteStateManager) ListResources(ctx context.Context, clusterName string) ([]*Resource, error) {
	panic("unimplemented")
}

// Migrate implements StateManager.
func (s *SQLiteStateManager) Migrate(ctx context.Context, target StateManager) error {
	panic("unimplemented")
}

// ReleaseLock implements StateManager.
func (s *SQLiteStateManager) ReleaseLock(ctx context.Context, lock Lock) error {
	panic("unimplemented")
}

// SaveClusterState implements StateManager.
func (s *SQLiteStateManager) SaveClusterState(ctx context.Context, state *ClusterState) error {
	panic("unimplemented")
}

// SaveResource implements StateManager.
func (s *SQLiteStateManager) SaveResource(ctx context.Context, resource *Resource) error {
	panic("unimplemented")
}

// Validate implements StateManager.
func (s *SQLiteStateManager) Validate(ctx context.Context) error {
	panic("unimplemented")
}

var _ StateManager = (*SQLiteStateManager)(nil)
