CREATE TABLE IF NOT EXISTS mcp_services (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id  BIGINT NOT NULL,
    name       VARCHAR(100) NOT NULL,
    transport  VARCHAR(16) NOT NULL CHECK (transport IN ('stdio', 'http')),
    endpoint   TEXT NOT NULL DEFAULT '',
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_mcp_services_tenant_enabled
    ON mcp_services (tenant_id, enabled);

ALTER TABLE mcp_services ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_mcp_services_tenant ON mcp_services;
CREATE POLICY sdpivot_mcp_services_tenant ON mcp_services
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
