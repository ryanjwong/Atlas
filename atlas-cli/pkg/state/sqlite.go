package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const sqliteStateSchema string = `
CREATE TABLE IF NOT EXISTS clusters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    provider TEXT NOT NULL,
    region TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unknown',
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

CREATE TABLE IF NOT EXISTS operation_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_name TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    operation_status TEXT NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    user_id TEXT NOT NULL,
    operation_details TEXT,
    error_message TEXT,
    metadata TEXT,
    FOREIGN KEY (cluster_name) REFERENCES clusters(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_clusters_name ON clusters(name);
CREATE INDEX IF NOT EXISTS idx_clusters_provider ON clusters(provider);
CREATE INDEX IF NOT EXISTS idx_resources_cluster_id ON cluster_resources(cluster_id);
CREATE INDEX IF NOT EXISTS idx_locks_expires_at ON state_locks(expires_at);
CREATE INDEX IF NOT EXISTS idx_operation_history_cluster ON operation_history(cluster_name);
CREATE INDEX IF NOT EXISTS idx_operation_history_type ON operation_history(operation_type);
CREATE INDEX IF NOT EXISTS idx_operation_history_status ON operation_history(operation_status);
CREATE INDEX IF NOT EXISTS idx_operation_history_time ON operation_history(started_at);
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
	return s.db.PingContext(ctx)
}

// DeleteClusterState implements StateManager.
func (s *SQLiteStateManager) DeleteClusterState(ctx context.Context, name string) error {
	query := `DELETE FROM clusters WHERE name = ?`
	result, err := s.db.ExecContext(ctx, query, name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows // Cluster not found
	}

	return nil
}

// DeleteResource implements StateManager.
func (s *SQLiteStateManager) DeleteResource(ctx context.Context, clusterName string, resourceID string) error {
	query := `DELETE FROM cluster_resources WHERE cluster_id = ? AND id = ?`
	result, err := s.db.ExecContext(ctx, query, clusterName, resourceID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Disconnect implements StateManager.
func (s *SQLiteStateManager) Disconnect(ctx context.Context) error {
	return s.db.Close()
}

// GetClusterState implements StateManager.
func (s *SQLiteStateManager) GetClusterState(ctx context.Context, name string) (*ClusterState, error) {
	query := `
		SELECT 
			id, name, provider, region, status, node_count, version,
			config, metadata, created_at, updated_at, created_by
		FROM clusters 
		WHERE name = ?
	`

	var cluster ClusterState
	var configJSON, metadataJSON string
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&cluster.ID,
		&cluster.Name,
		&cluster.Provider,
		&cluster.Region,
		&cluster.Status,
		&cluster.NodeCount,
		&cluster.Version,
		&configJSON,
		&metadataJSON,
		&createdAt,
		&updatedAt,
		&cluster.CreatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(configJSON), &cluster.Config); err != nil {
		cluster.Config = make(map[string]interface{})
	}
	if err := json.Unmarshal([]byte(metadataJSON), &cluster.Metadata); err != nil {
		cluster.Metadata = make(map[string]string)
	}

	cluster.CreatedAt = createdAt
	cluster.UpdatedAt = updatedAt

	resources, err := s.ListResources(ctx, name)
	if err != nil {
		return nil, err
	}
	cluster.Resources = resources

	return &cluster, nil
}

// GetResource implements StateManager.
func (s *SQLiteStateManager) GetResource(ctx context.Context, clusterName string, resourceID string) (*Resource, error) {
	query := `
		SELECT id, cluster_id, resource_type, resource_name, namespace, status,
			   config, dependencies, created_at, updated_at
		FROM cluster_resources 
		WHERE cluster_id = ? AND id = ?
	`

	var resource Resource
	var configJSON, dependenciesJSON sql.NullString
	var namespace sql.NullString
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx, query, clusterName, resourceID).Scan(
		&resource.ID,
		&resource.ClusterName,
		&resource.Type,
		&resource.Name,
		&namespace,
		&resource.Status,
		&configJSON,
		&dependenciesJSON,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	resource.Namespace = namespace.String
	resource.CreatedAt = createdAt
	resource.UpdatedAt = updatedAt

	if configJSON.Valid {
		if err := json.Unmarshal([]byte(configJSON.String), &resource.Config); err != nil {
			resource.Config = make(map[string]any)
		}
	} else {
		resource.Config = make(map[string]any)
	}

	if dependenciesJSON.Valid {
		if err := json.Unmarshal([]byte(dependenciesJSON.String), &resource.Dependencies); err != nil {
			resource.Dependencies = []string{}
		}
	} else {
		resource.Dependencies = []string{}
	}

	return &resource, nil
}

// Health implements StateManager.
func (s *SQLiteStateManager) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// ListClusters implements StateManager.
func (s *SQLiteStateManager) ListClusters(ctx context.Context) ([]*ClusterState, error) {
	query := `
		SELECT 
			id, name, provider, region, status, node_count, version,
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
			&cluster.Status,
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
	query := `
		SELECT id, cluster_id, resource_type, resource_name, namespace, status,
			   config, dependencies, created_at, updated_at
		FROM cluster_resources 
		WHERE cluster_id = ?
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, clusterName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*Resource

	for rows.Next() {
		var resource Resource
		var configJSON, dependenciesJSON sql.NullString
		var namespace sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&resource.ID,
			&resource.ClusterName,
			&resource.Type,
			&resource.Name,
			&namespace,
			&resource.Status,
			&configJSON,
			&dependenciesJSON,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, err
		}

		resource.Namespace = namespace.String
		resource.CreatedAt = createdAt
		resource.UpdatedAt = updatedAt

		if configJSON.Valid {
			if err := json.Unmarshal([]byte(configJSON.String), &resource.Config); err != nil {
				resource.Config = make(map[string]any)
			}
		} else {
			resource.Config = make(map[string]any)
		}

		if dependenciesJSON.Valid {
			if err := json.Unmarshal([]byte(dependenciesJSON.String), &resource.Dependencies); err != nil {
				resource.Dependencies = []string{}
			}
		} else {
			resource.Dependencies = []string{}
		}

		resources = append(resources, &resource)
	}

	return resources, rows.Err()
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
	configJSON, err := json.Marshal(state.Config)
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(state.Metadata)
	if err != nil {
		return err
	}

	if state.ID == 0 {
		query := `
			INSERT INTO clusters (name, provider, region, status, node_count, version, config, metadata, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		result, err := s.db.ExecContext(ctx, query,
			state.Name, state.Provider, state.Region, state.Status, state.NodeCount, state.Version,
			string(configJSON), string(metadataJSON), state.CreatedBy)
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		state.ID = int(id)
	} else {
		query := `
			UPDATE clusters 
			SET provider = ?, region = ?, status = ?, node_count = ?, version = ?, config = ?, 
				metadata = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`
		_, err := s.db.ExecContext(ctx, query,
			state.Provider, state.Region, state.Status, state.NodeCount, state.Version,
			string(configJSON), string(metadataJSON), state.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// SaveResource implements StateManager.
func (s *SQLiteStateManager) SaveResource(ctx context.Context, resource *Resource) error {
	configJSON, err := json.Marshal(resource.Config)
	if err != nil {
		return err
	}

	dependenciesJSON, err := json.Marshal(resource.Dependencies)
	if err != nil {
		return err
	}

	query := `
		INSERT OR REPLACE INTO cluster_resources 
		(id, cluster_id, resource_type, resource_name, namespace, status, config, dependencies, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err = s.db.ExecContext(ctx, query,
		resource.ID, resource.ClusterName, resource.Type, resource.Name,
		resource.Namespace, resource.Status, string(configJSON), string(dependenciesJSON))

	return err
}

// Validate implements StateManager.
func (s *SQLiteStateManager) Validate(ctx context.Context) error {
	// Check database connection
	if err := s.Health(ctx); err != nil {
		return err
	}

	// Verify table structure exists
	tables := []string{"clusters", "cluster_resources", "state_locks", "operation_history"}
	for _, table := range tables {
		query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
		var name string
		if err := s.db.QueryRowContext(ctx, query, table).Scan(&name); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("table %s does not exist", table)
			}
			return err
		}
	}

	return nil
}

// StartOperation implements StateManager.
func (s *SQLiteStateManager) StartOperation(ctx context.Context, op *OperationHistory) error {
	detailsJSON, err := json.Marshal(op.OperationDetails)
	if err != nil {
		return err
	}

	metadataJSON, err := json.Marshal(op.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO operation_history 
		(cluster_name, operation_type, operation_status, user_id, operation_details, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query,
		op.ClusterName, string(op.OperationType), string(op.OperationStatus),
		op.UserID, string(detailsJSON), string(metadataJSON))
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	op.ID = int(id)

	return nil
}

// UpdateOperation implements StateManager.
func (s *SQLiteStateManager) UpdateOperation(ctx context.Context, id int, status OperationStatus, errorMsg string) error {
	query := `
		UPDATE operation_history 
		SET operation_status = ?, error_message = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, string(status), errorMsg, id)
	return err
}

// CompleteOperation implements StateManager.
func (s *SQLiteStateManager) CompleteOperation(ctx context.Context, id int, status OperationStatus) error {
	query := `
		UPDATE operation_history 
		SET operation_status = ?, completed_at = CURRENT_TIMESTAMP,
			duration_ms = (julianday('now') - julianday(started_at)) * 24 * 60 * 60 * 1000
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, string(status), id)
	return err
}

// GetOperationHistory implements StateManager.
func (s *SQLiteStateManager) GetOperationHistory(ctx context.Context, clusterName string, limit int) ([]*OperationHistory, error) {
	query := `
		SELECT id, cluster_name, operation_type, operation_status, started_at, completed_at,
			   duration_ms, user_id, operation_details, error_message, metadata
		FROM operation_history 
		WHERE cluster_name = ?
		ORDER BY started_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, clusterName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operations []*OperationHistory

	for rows.Next() {
		var op OperationHistory
		var detailsJSON, metadataJSON sql.NullString
		var completedAt sql.NullTime
		var durationMS sql.NullFloat64
		var startedAt time.Time
		var errorMessage sql.NullString
		var opType, opStatus string

		err := rows.Scan(
			&op.ID, &op.ClusterName, &opType, &opStatus, &startedAt, &completedAt,
			&durationMS, &op.UserID, &detailsJSON, &errorMessage, &metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		op.OperationType = OperationType(opType)
		op.OperationStatus = OperationStatus(opStatus)
		op.StartedAt = startedAt

		if completedAt.Valid {
			op.CompletedAt = &completedAt.Time
		}
		if durationMS.Valid {
			op.DurationMS = &durationMS.Float64
		}

		if detailsJSON.Valid && detailsJSON.String != "" {
			if err := json.Unmarshal([]byte(detailsJSON.String), &op.OperationDetails); err != nil {
				op.OperationDetails = make(map[string]interface{})
			}
		} else {
			op.OperationDetails = make(map[string]interface{})
		}

		if errorMessage.Valid {
			op.ErrorMessage = errorMessage.String
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &op.Metadata); err != nil {
				op.Metadata = make(map[string]string)
			}
		} else {
			op.Metadata = make(map[string]string)
		}

		operations = append(operations, &op)
	}

	return operations, rows.Err()
}

// GetOperation implements StateManager.
func (s *SQLiteStateManager) GetOperation(ctx context.Context, id int) (*OperationHistory, error) {
	query := `
		SELECT id, cluster_name, operation_type, operation_status, started_at, completed_at,
			   duration_ms, user_id, operation_details, error_message, metadata
		FROM operation_history 
		WHERE id = ?
	`

	var op OperationHistory
	var detailsJSON, metadataJSON sql.NullString
	var completedAt sql.NullTime
	var durationMS sql.NullFloat64
	var startedAt time.Time
	var opType, opStatus string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&op.ID, &op.ClusterName, &opType, &opStatus, &startedAt, &completedAt,
		&durationMS, &op.UserID, &detailsJSON, &op.ErrorMessage, &metadataJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	op.OperationType = OperationType(opType)
	op.OperationStatus = OperationStatus(opStatus)
	op.StartedAt = startedAt

	if completedAt.Valid {
		op.CompletedAt = &completedAt.Time
	}
	if durationMS.Valid {
		op.DurationMS = &durationMS.Float64
	}

	if detailsJSON.Valid && detailsJSON.String != "" {
		if err := json.Unmarshal([]byte(detailsJSON.String), &op.OperationDetails); err != nil {
			op.OperationDetails = make(map[string]interface{})
		}
	} else {
		op.OperationDetails = make(map[string]interface{})
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &op.Metadata); err != nil {
			op.Metadata = make(map[string]string)
		}
	} else {
		op.Metadata = make(map[string]string)
	}

	return &op, nil
}

var _ StateManager = (*SQLiteStateManager)(nil)
