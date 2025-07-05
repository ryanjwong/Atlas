-- Atlas CLI SQLite State Schema

-- Clusters table - stores cluster configurations and state
CREATE TABLE IF NOT EXISTS clusters (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    provider TEXT NOT NULL,
    region TEXT NOT NULL,
    status TEXT NOT NULL,
    node_count INTEGER NOT NULL DEFAULT 3,
    version TEXT NOT NULL,
    config TEXT NOT NULL, -- JSON configuration
    metadata TEXT, -- JSON metadata
    credentials BLOB, -- Encrypted credentials
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL DEFAULT 'atlas-cli'
);

-- Resources table - stores individual resources within clusters
CREATE TABLE IF NOT EXISTS cluster_resources (
    id TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_name TEXT NOT NULL,
    namespace TEXT,
    status TEXT NOT NULL,
    config TEXT NOT NULL, -- JSON configuration
    dependencies TEXT, -- JSON array of resource IDs
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE,
    UNIQUE(cluster_id, resource_type, resource_name, namespace)
);

-- State locks table - manages distributed locking
CREATE TABLE IF NOT EXISTS state_locks (
    resource_name TEXT PRIMARY KEY,
    lock_id TEXT NOT NULL,
    acquired_by TEXT NOT NULL,
    acquired_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    metadata TEXT -- JSON metadata
);

-- Backups table - tracks backup metadata
CREATE TABLE IF NOT EXISTS backups (
    id TEXT PRIMARY KEY,
    backup_type TEXT NOT NULL, -- full, incremental, differential
    cluster_name TEXT,
    components TEXT NOT NULL, -- JSON array of components
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    size_bytes INTEGER NOT NULL DEFAULT 0,
    checksum TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    location TEXT NOT NULL,
    parent_id TEXT, -- For incremental backups
    tags TEXT, -- JSON object
    metadata TEXT, -- JSON metadata
    FOREIGN KEY (parent_id) REFERENCES backups(id)
);

-- Audit log table - tracks all operations
CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    operation TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    before_state TEXT, -- JSON
    after_state TEXT, -- JSON
    metadata TEXT, -- JSON
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_clusters_name ON clusters(name);
CREATE INDEX IF NOT EXISTS idx_clusters_provider ON clusters(provider);
CREATE INDEX IF NOT EXISTS idx_clusters_status ON clusters(status);
CREATE INDEX IF NOT EXISTS idx_clusters_created_at ON clusters(created_at);

CREATE INDEX IF NOT EXISTS idx_resources_cluster_id ON cluster_resources(cluster_id);
CREATE INDEX IF NOT EXISTS idx_resources_type ON cluster_resources(resource_type);
CREATE INDEX IF NOT EXISTS idx_resources_status ON cluster_resources(status);

CREATE INDEX IF NOT EXISTS idx_locks_expires_at ON state_locks(expires_at);

CREATE INDEX IF NOT EXISTS idx_backups_cluster ON backups(cluster_name);
CREATE INDEX IF NOT EXISTS idx_backups_created_at ON backups(created_at);
CREATE INDEX IF NOT EXISTS idx_backups_status ON backups(status);

CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_log(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id);

-- Triggers to update updated_at timestamps
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

-- View for cluster summary
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