-- Atlas CLI SQLite Queries

-- =============================================================================
-- CLUSTER OPERATIONS
-- =============================================================================

-- name: SaveClusterState
-- Save or update cluster state
INSERT OR REPLACE INTO clusters (
    id, name, provider, region, status, node_count, version, 
    config, metadata, credentials, created_by, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);

-- name: GetClusterState
-- Get cluster state by name
SELECT 
    id, name, provider, region, status, node_count, version,
    config, metadata, credentials, created_at, updated_at, created_by
FROM clusters 
WHERE name = ?;

-- name: GetClusterStateByID
-- Get cluster state by ID
SELECT 
    id, name, provider, region, status, node_count, version,
    config, metadata, credentials, created_at, updated_at, created_by
FROM clusters 
WHERE id = ?;

-- name: ListClusters
-- List all clusters with optional filters
SELECT 
    id, name, provider, region, status, node_count, version,
    config, metadata, created_at, updated_at, created_by
FROM clusters 
WHERE (? = '' OR provider = ?)
  AND (? = '' OR status = ?)
ORDER BY created_at DESC;

-- name: DeleteClusterState
-- Delete cluster and all associated resources
DELETE FROM clusters WHERE name = ?;

-- name: UpdateClusterStatus
-- Update only cluster status
UPDATE clusters 
SET status = ?, updated_at = CURRENT_TIMESTAMP 
WHERE name = ?;

-- name: ClusterExists
-- Check if cluster exists
SELECT COUNT(*) FROM clusters WHERE name = ?;

-- =============================================================================
-- RESOURCE OPERATIONS
-- =============================================================================

-- name: SaveResource
-- Save or update cluster resource
INSERT OR REPLACE INTO cluster_resources (
    id, cluster_id, resource_type, resource_name, namespace, 
    status, config, dependencies, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);

-- name: GetResource
-- Get resource by cluster and resource ID
SELECT 
    id, cluster_id, resource_type, resource_name, namespace,
    status, config, dependencies, created_at, updated_at
FROM cluster_resources 
WHERE cluster_id = ? AND id = ?;

-- name: ListResources
-- List all resources for a cluster
SELECT 
    id, cluster_id, resource_type, resource_name, namespace,
    status, config, dependencies, created_at, updated_at
FROM cluster_resources 
WHERE cluster_id = ?
ORDER BY resource_type, resource_name;

-- name: ListResourcesByType
-- List resources by type for a cluster
SELECT 
    id, cluster_id, resource_type, resource_name, namespace,
    status, config, dependencies, created_at, updated_at
FROM cluster_resources 
WHERE cluster_id = ? AND resource_type = ?
ORDER BY resource_name;

-- name: DeleteResource
-- Delete a specific resource
DELETE FROM cluster_resources 
WHERE cluster_id = ? AND id = ?;

-- name: DeleteClusterResources
-- Delete all resources for a cluster
DELETE FROM cluster_resources WHERE cluster_id = ?;

-- name: UpdateResourceStatus
-- Update resource status
UPDATE cluster_resources 
SET status = ?, updated_at = CURRENT_TIMESTAMP 
WHERE cluster_id = ? AND id = ?;

-- =============================================================================
-- LOCKING OPERATIONS
-- =============================================================================

-- name: AcquireLock
-- Acquire a distributed lock
INSERT INTO state_locks (resource_name, lock_id, acquired_by, expires_at, metadata)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(resource_name) DO UPDATE SET
    lock_id = excluded.lock_id,
    acquired_by = excluded.acquired_by,
    acquired_at = CURRENT_TIMESTAMP,
    expires_at = excluded.expires_at,
    metadata = excluded.metadata
WHERE expires_at < CURRENT_TIMESTAMP;

-- name: ReleaseLock
-- Release a distributed lock
DELETE FROM state_locks 
WHERE resource_name = ? AND lock_id = ?;

-- name: RefreshLock
-- Refresh lock expiration
UPDATE state_locks 
SET expires_at = ?, acquired_at = CURRENT_TIMESTAMP 
WHERE resource_name = ? AND lock_id = ?;

-- name: GetLock
-- Get lock information
SELECT 
    resource_name, lock_id, acquired_by, acquired_at, expires_at, metadata
FROM state_locks 
WHERE resource_name = ?;

-- name: CleanupExpiredLocks
-- Remove expired locks
DELETE FROM state_locks WHERE expires_at < CURRENT_TIMESTAMP;

-- name: ListActiveLocks
-- List all active locks
SELECT 
    resource_name, lock_id, acquired_by, acquired_at, expires_at, metadata
FROM state_locks 
WHERE expires_at > CURRENT_TIMESTAMP
ORDER BY acquired_at;

-- =============================================================================
-- BACKUP OPERATIONS
-- =============================================================================

-- name: SaveBackup
-- Save backup metadata
INSERT INTO backups (
    id, backup_type, cluster_name, components, size_bytes, 
    checksum, status, location, parent_id, tags, metadata
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetBackup
-- Get backup by ID
SELECT 
    id, backup_type, cluster_name, components, created_at, size_bytes,
    checksum, status, location, parent_id, tags, metadata
FROM backups 
WHERE id = ?;

-- name: ListBackups
-- List backups with optional filters
SELECT 
    id, backup_type, cluster_name, components, created_at, size_bytes,
    checksum, status, location, parent_id, tags, metadata
FROM backups 
WHERE (? = '' OR cluster_name = ?)
  AND (? = '' OR backup_type = ?)
  AND (? = '' OR status = ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateBackupStatus
-- Update backup status
UPDATE backups 
SET status = ?, metadata = ? 
WHERE id = ?;

-- name: DeleteBackup
-- Delete backup metadata
DELETE FROM backups WHERE id = ?;

-- name: GetBackupsByCluster
-- Get all backups for a cluster
SELECT 
    id, backup_type, cluster_name, components, created_at, size_bytes,
    checksum, status, location, parent_id, tags, metadata
FROM backups 
WHERE cluster_name = ?
ORDER BY created_at DESC;

-- name: CleanupOldBackups
-- Delete backups older than specified date
DELETE FROM backups 
WHERE created_at < ? 
  AND (? = '' OR cluster_name = ?);

-- =============================================================================
-- AUDIT LOG OPERATIONS
-- =============================================================================

-- name: LogOperation
-- Log an operation to audit trail
INSERT INTO audit_log (
    id, operation, resource_type, resource_id, user_id,
    before_state, after_state, metadata, success, error_message
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAuditLog
-- Get audit log entries with filters
SELECT 
    id, operation, resource_type, resource_id, user_id, timestamp,
    before_state, after_state, metadata, success, error_message
FROM audit_log 
WHERE (? = '' OR resource_type = ?)
  AND (? = '' OR resource_id = ?)
  AND (? = '' OR user_id = ?)
  AND timestamp >= ?
  AND timestamp <= ?
ORDER BY timestamp DESC
LIMIT ? OFFSET ?;

-- name: CleanupAuditLog
-- Clean up old audit log entries
DELETE FROM audit_log 
WHERE timestamp < ?;

-- =============================================================================
-- MAINTENANCE OPERATIONS
-- =============================================================================

-- name: GetDatabaseInfo
-- Get database statistics
SELECT 
    (SELECT COUNT(*) FROM clusters) as cluster_count,
    (SELECT COUNT(*) FROM cluster_resources) as resource_count,
    (SELECT COUNT(*) FROM state_locks WHERE expires_at > CURRENT_TIMESTAMP) as active_locks,
    (SELECT COUNT(*) FROM backups) as backup_count,
    (SELECT COUNT(*) FROM audit_log) as audit_entries;

-- name: VacuumDatabase
-- Optimize database
VACUUM;

-- name: GetTableSizes
-- Get table sizes for monitoring
SELECT 
    name as table_name,
    COUNT(*) as row_count
FROM (
    SELECT 'clusters' as name FROM clusters
    UNION ALL
    SELECT 'cluster_resources' as name FROM cluster_resources
    UNION ALL
    SELECT 'state_locks' as name FROM state_locks
    UNION ALL
    SELECT 'backups' as name FROM backups
    UNION ALL
    SELECT 'audit_log' as name FROM audit_log
) 
GROUP BY name;

-- name: CheckIntegrity
-- Check database integrity
PRAGMA integrity_check;

-- =============================================================================
-- MIGRATION OPERATIONS
-- =============================================================================

-- name: GetSchemaVersion
-- Get current schema version
PRAGMA user_version;

-- name: SetSchemaVersion
-- Set schema version
PRAGMA user_version = ?;

-- name: ListTables
-- List all tables
SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;