package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Example implementation showing how to use the SQL queries

type SQLiteStateManager struct {
	db *sql.DB
}

func NewSQLiteStateManager(dbPath string) (*SQLiteStateManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	
	if err := db.Ping(); err != nil {
		return nil, err
	}
	
	manager := &SQLiteStateManager{db: db}
	
	// Run migrations
	if err := manager.runMigrations(); err != nil {
		return nil, err
	}
	
	return manager, nil
}

// Example: Save cluster state
func (s *SQLiteStateManager) SaveClusterState(ctx context.Context, cluster *ClusterState) error {
	configJSON, _ := json.Marshal(cluster.Config)
	metadataJSON, _ := json.Marshal(cluster.Metadata)
	
	query := `
		INSERT OR REPLACE INTO clusters (
			id, name, provider, region, status, node_count, version, 
			config, metadata, credentials, created_by, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	
	_, err := s.db.ExecContext(ctx, query,
		cluster.ID,
		cluster.Name,
		cluster.Provider,
		cluster.Region,
		cluster.Status,
		cluster.NodeCount,
		cluster.Version,
		string(configJSON),
		string(metadataJSON),
		cluster.EncryptedCredentials,
		cluster.CreatedBy,
	)
	
	return err
}

// Example: Get cluster state
func (s *SQLiteStateManager) GetClusterState(ctx context.Context, name string) (*ClusterState, error) {
	query := `
		SELECT 
			id, name, provider, region, status, node_count, version,
			config, metadata, credentials, created_at, updated_at, created_by
		FROM clusters 
		WHERE name = ?`
	
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
		&cluster.EncryptedCredentials,
		&createdAt,
		&updatedAt,
		&cluster.CreatedBy,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cluster not found: %s", name)
		}
		return nil, err
	}
	
	// Parse JSON fields
	json.Unmarshal([]byte(configJSON), &cluster.Config)
	json.Unmarshal([]byte(metadataJSON), &cluster.Metadata)
	
	cluster.CreatedAt = createdAt
	cluster.UpdatedAt = updatedAt
	
	return &cluster, nil
}

// Example: List clusters with filters
func (s *SQLiteStateManager) ListClusters(ctx context.Context, provider, status string) ([]*ClusterState, error) {
	query := `
		SELECT 
			id, name, provider, region, status, node_count, version,
			config, metadata, created_at, updated_at, created_by
		FROM clusters 
		WHERE (? = '' OR provider = ?)
		  AND (? = '' OR status = ?)
		ORDER BY created_at DESC`
	
	rows, err := s.db.QueryContext(ctx, query, provider, provider, status, status)
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
		
		// Parse JSON fields
		json.Unmarshal([]byte(configJSON), &cluster.Config)
		json.Unmarshal([]byte(metadataJSON), &cluster.Metadata)
		
		cluster.CreatedAt = createdAt
		cluster.UpdatedAt = updatedAt
		
		clusters = append(clusters, &cluster)
	}
	
	return clusters, rows.Err()
}

// Example: Acquire lock
func (s *SQLiteStateManager) AcquireLock(ctx context.Context, resource, lockID, acquiredBy string, expiresAt time.Time) error {
	query := `
		INSERT INTO state_locks (resource_name, lock_id, acquired_by, expires_at, metadata)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(resource_name) DO UPDATE SET
			lock_id = excluded.lock_id,
			acquired_by = excluded.acquired_by,
			acquired_at = CURRENT_TIMESTAMP,
			expires_at = excluded.expires_at,
			metadata = excluded.metadata
		WHERE expires_at < CURRENT_TIMESTAMP`
	
	result, err := s.db.ExecContext(ctx, query, resource, lockID, acquiredBy, expiresAt, "{}")
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("failed to acquire lock for resource: %s", resource)
	}
	
	return nil
}

// Example: Cleanup expired locks
func (s *SQLiteStateManager) CleanupExpiredLocks(ctx context.Context) (int64, error) {
	query := `DELETE FROM state_locks WHERE expires_at < CURRENT_TIMESTAMP`
	
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected()
}

// Example: Log operation to audit trail
func (s *SQLiteStateManager) LogOperation(ctx context.Context, operation, resourceType, resourceID, userID string, beforeState, afterState interface{}, success bool, errorMsg string) error {
	beforeJSON, _ := json.Marshal(beforeState)
	afterJSON, _ := json.Marshal(afterState)
	
	query := `
		INSERT INTO audit_log (
			id, operation, resource_type, resource_id, user_id,
			before_state, after_state, metadata, success, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := s.db.ExecContext(ctx, query,
		generateID(),
		operation,
		resourceType,
		resourceID,
		userID,
		string(beforeJSON),
		string(afterJSON),
		"{}",
		success,
		errorMsg,
	)
	
	return err
}

// Example: Run database migrations
func (s *SQLiteStateManager) runMigrations() error {
	// Get current schema version
	var version int
	err := s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return err
	}
	
	// Run migrations based on current version
	if version < 1 {
		if err := s.runMigration001(); err != nil {
			return err
		}
	}
	
	if version < 2 {
		if err := s.runMigration002(); err != nil {
			return err
		}
	}
	
	// Add more migrations as needed...
	
	return nil
}

func (s *SQLiteStateManager) runMigration001() error {
	// Execute the initial schema creation
	// This would read from schema.sql or embed the SQL
	return nil
}

func (s *SQLiteStateManager) runMigration002() error {
	// Execute backup table creation
	return nil
}

// Helper types (these would be in your actual state package)
type ClusterState struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Provider             string                 `json:"provider"`
	Region               string                 `json:"region"`
	Status               string                 `json:"status"`
	NodeCount            int                    `json:"nodeCount"`
	Version              string                 `json:"version"`
	Config               map[string]interface{} `json:"config"`
	Metadata             map[string]interface{} `json:"metadata"`
	EncryptedCredentials []byte                 `json:"-"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
	CreatedBy            string                 `json:"createdBy"`
}

func generateID() string {
	// Generate UUID or similar
	return fmt.Sprintf("id-%d", time.Now().UnixNano())
}

// Usage example:
func ExampleUsage() {
	ctx := context.Background()
	
	// Initialize state manager
	stateManager, err := NewSQLiteStateManager("atlas.db")
	if err != nil {
		panic(err)
	}
	
	// Create a cluster
	cluster := &ClusterState{
		ID:        "cluster-123",
		Name:      "my-cluster",
		Provider:  "aws",
		Region:    "us-west-2",
		Status:    "creating",
		NodeCount: 3,
		Version:   "1.28",
		Config:    map[string]interface{}{"instanceType": "t3.medium"},
		Metadata:  map[string]interface{}{"environment": "dev"},
		CreatedBy: "user@example.com",
	}
	
	// Save cluster
	err = stateManager.SaveClusterState(ctx, cluster)
	if err != nil {
		panic(err)
	}
	
	// List clusters
	clusters, err := stateManager.ListClusters(ctx, "", "")
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Found %d clusters\n", len(clusters))
}