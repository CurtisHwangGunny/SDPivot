CREATE TABLE IF NOT EXISTS third_party_connector (
    id           VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id    BIGINT NOT NULL,
    name         VARCHAR(100) NOT NULL,
    type         VARCHAR(32) NOT NULL CHECK (type IN ('users', 'departments', 'documents')),
    base_url     TEXT NOT NULL,
    auth_type    VARCHAR(16) NOT NULL DEFAULT 'none' CHECK (auth_type IN ('none', 'bearer', 'basic')),
    auth_token   TEXT NOT NULL DEFAULT '',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    sync_rule    JSONB NOT NULL DEFAULT '{}'::JSONB,
    last_sync_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_third_party_connector_tenant_type
    ON third_party_connector (tenant_id, type, enabled);

ALTER TABLE third_party_connector ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_third_party_connector_tenant ON third_party_connector;
CREATE POLICY sdpivot_third_party_connector_tenant ON third_party_connector
    FOR ALL USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());
