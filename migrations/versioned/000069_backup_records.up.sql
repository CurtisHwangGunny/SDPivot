CREATE TABLE IF NOT EXISTS backup_records (
    id VARCHAR(36) PRIMARY KEY,
    status VARCHAR(20) NOT NULL,
    trigger VARCHAR(20) NOT NULL,
    database_driver VARCHAR(20) NOT NULL,
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL DEFAULT '',
    format VARCHAR(20) NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    checksum_sha256 CHAR(64) NOT NULL DEFAULT '',
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_backup_records_status ON backup_records(status);
CREATE INDEX IF NOT EXISTS idx_backup_records_created_at ON backup_records(created_at DESC);
