-- SDPivot deployment update and rollback history.

CREATE TABLE IF NOT EXISTS system_update_log (
    id           VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id    BIGINT NOT NULL,
    from_version VARCHAR(64) NOT NULL DEFAULT '',
    to_version   VARCHAR(64) NOT NULL DEFAULT '',
    status       VARCHAR(20) NOT NULL DEFAULT 'success'
                 CHECK (status IN ('pending', 'running', 'success', 'failed')),
    update_type  VARCHAR(20) NOT NULL DEFAULT 'online'
                 CHECK (update_type IN ('online', 'offline', 'rollback')),
    content      TEXT NOT NULL DEFAULT '',
    created_by   VARCHAR(36) NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_system_update_log_tenant_time
    ON system_update_log (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_update_log_tenant_status
    ON system_update_log (tenant_id, status);

ALTER TABLE system_update_log ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000022_system_update_log ON system_update_log;
CREATE POLICY sdpivot_op_bootstrap_000022_system_update_log ON system_update_log
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
