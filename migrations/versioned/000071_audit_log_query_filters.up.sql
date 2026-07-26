-- Optimize the tenant-scoped audit query API's time-range scans.
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created_at
    ON audit_logs (tenant_id, created_at DESC);
