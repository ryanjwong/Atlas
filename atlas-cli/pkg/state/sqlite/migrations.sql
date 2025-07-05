-- Atlas CLI SQLite Migrations

-- =============================================================================
-- MIGRATION 001: Initial Schema
-- =============================================================================

-- Migration: 001_initial_schema.sql
-- Description: Create initial tables and indexes
-- Version: 1

BEGIN TRANSACTION;

-- Create clusters table
CREATE TABLE IF NOT EXISTS clusters (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    provider TEXT NOT NULL,
    region TEXT NOT NULL,
    status TEXT NOT NULL,
    node_count INTEGER NOT NULL DEFAULT 3,
    version TEXT NOT NULL,
    config TEXT NOT NULL,
    metadata TEXT,
    credentials BLOB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL DEFAULT 'atlas-cli'
);

-- Create resources table
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

-- Create locks table
CREATE TABLE IF NOT EXISTS state_locks (
    resource_name TEXT PRIMARY KEY,
    lock_id TEXT NOT NULL,
    acquired_by TEXT NOT NULL,
    acquired_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    metadata TEXT
);

-- Create basic indexes
CREATE INDEX IF NOT EXISTS idx_clusters_name ON clusters(name);
CREATE INDEX IF NOT EXISTS idx_clusters_provider ON clusters(provider);
CREATE INDEX IF NOT EXISTS idx_resources_cluster_id ON cluster_resources(cluster_id);
CREATE INDEX IF NOT EXISTS idx_locks_expires_at ON state_locks(expires_at);

-- Update schema version
PRAGMA user_version = 1;

COMMIT;

-- =============================================================================
-- MIGRATION 002: Add Backup Support
-- =============================================================================

-- Migration: 002_add_backups.sql
-- Description: Add backup tracking tables
-- Version: 2

BEGIN TRANSACTION;

-- Create backups table
CREATE TABLE IF NOT EXISTS backups (
    id TEXT PRIMARY KEY,
    backup_type TEXT NOT NULL,
    cluster_name TEXT,
    components TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    size_bytes INTEGER NOT NULL DEFAULT 0,
    checksum TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    location TEXT NOT NULL,
    parent_id TEXT,
    tags TEXT,
    metadata TEXT,
    FOREIGN KEY (parent_id) REFERENCES backups(id)
);

-- Create backup indexes
CREATE INDEX IF NOT EXISTS idx_backups_cluster ON backups(cluster_name);
CREATE INDEX IF NOT EXISTS idx_backups_created_at ON backups(created_at);
CREATE INDEX IF NOT EXISTS idx_backups_status ON backups(status);

-- Update schema version
PRAGMA user_version = 2;

COMMIT;

-- =============================================================================
-- MIGRATION 003: Add Audit Logging
-- =============================================================================

-- Migration: 003_add_audit_log.sql
-- Description: Add audit logging capability
-- Version: 3

BEGIN TRANSACTION;

-- Create audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    operation TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    before_state TEXT,
    after_state TEXT,
    metadata TEXT,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT
);

-- Create audit indexes
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_log(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id);

-- Update schema version
PRAGMA user_version = 3;

COMMIT;

-- =============================================================================
-- MIGRATION 004: Add Triggers and Views
-- =============================================================================

-- Migration: 004_add_triggers_views.sql
-- Description: Add update triggers and helpful views
-- Version: 4

BEGIN TRANSACTION;

-- Create update triggers
CREATE TRIGGER IF NOT EXISTS update_clusters_timestamp 
    AFTER UPDATE ON clusters
    BEGIN
        UPDATE clusters SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS update_resources_timestamp 
    AFTER UPDATE ON cluster_resources
    BEGIN
        UPDATE cluster_resources SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

-- Create cluster summary view
CREATE VIEW IF NOT EXISTS cluster_summary AS
SELECT 
    c.id,
    c.name,
    c.provider,
    c.region,
    c.status,
    c.node_count,
    c.version,
    c.created_at,
    c.updated_at,
    COUNT(r.id) as resource_count,
    GROUP_CONCAT(DISTINCT r.resource_type) as resource_types
FROM clusters c
LEFT JOIN cluster_resources r ON c.id = r.cluster_id
GROUP BY c.id, c.name, c.provider, c.region, c.status, c.node_count, c.version, c.created_at, c.updated_at;

-- Update schema version
PRAGMA user_version = 4;

COMMIT;

-- =============================================================================
-- MIGRATION 005: Add Additional Indexes
-- =============================================================================

-- Migration: 005_add_indexes.sql
-- Description: Add performance indexes for common queries
-- Version: 5

BEGIN TRANSACTION;

-- Additional cluster indexes
CREATE INDEX IF NOT EXISTS idx_clusters_status ON clusters(status);
CREATE INDEX IF NOT EXISTS idx_clusters_created_at ON clusters(created_at);

-- Additional resource indexes
CREATE INDEX IF NOT EXISTS idx_resources_type ON cluster_resources(resource_type);
CREATE INDEX IF NOT EXISTS idx_resources_status ON cluster_resources(status);

-- Update schema version
PRAGMA user_version = 5;

COMMIT;

-- =============================================================================
-- ROLLBACK MIGRATIONS
-- =============================================================================

-- Rollback from version 5 to 4
-- DROP INDEX IF EXISTS idx_clusters_status;
-- DROP INDEX IF EXISTS idx_clusters_created_at;
-- DROP INDEX IF EXISTS idx_resources_type;
-- DROP INDEX IF EXISTS idx_resources_status;
-- PRAGMA user_version = 4;

-- Rollback from version 4 to 3
-- DROP TRIGGER IF EXISTS update_clusters_timestamp;
-- DROP TRIGGER IF EXISTS update_resources_timestamp;
-- DROP VIEW IF EXISTS cluster_summary;
-- PRAGMA user_version = 3;

-- Rollback from version 3 to 2
-- DROP TABLE IF EXISTS audit_log;
-- PRAGMA user_version = 2;

-- Rollback from version 2 to 1
-- DROP TABLE IF EXISTS backups;
-- PRAGMA user_version = 1;

-- Complete rollback (use with caution!)
-- DROP TABLE IF EXISTS audit_log;
-- DROP TABLE IF EXISTS backups;
-- DROP TABLE IF EXISTS cluster_resources;
-- DROP TABLE IF EXISTS state_locks;
-- DROP TABLE IF EXISTS clusters;
-- DROP VIEW IF EXISTS cluster_summary;
-- PRAGMA user_version = 0;