-- Enforce tenant isolation for document chunks through their owning document.
CREATE TABLE IF NOT EXISTS sdpivot_rls_migration_000012_state (
    table_name          TEXT PRIMARY KEY,
    rls_was_enabled     BOOLEAN NOT NULL,
    force_was_enabled   BOOLEAN NOT NULL
);

INSERT INTO sdpivot_rls_migration_000012_state (table_name, rls_was_enabled, force_was_enabled)
SELECT 'document_chunks', c.relrowsecurity, c.relforcerowsecurity
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname = 'document_chunks'
ON CONFLICT (table_name) DO NOTHING;

ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_policy
        WHERE polrelid = 'public.document_chunks'::regclass
          AND polname = 'document_chunks_tenant_isolation_000012'
    ) THEN
        CREATE POLICY document_chunks_tenant_isolation_000012 ON document_chunks
            FOR ALL
            USING (
                is_ops_admin_context()
                OR EXISTS (
                    SELECT 1
                    FROM documents d
                    WHERE d.id = document_chunks.document_id
                      AND d.tenant_id = document_chunks.tenant_id
                      AND d.tenant_id = get_current_tenant_id()
                )
            )
            WITH CHECK (
                is_ops_admin_context()
                OR EXISTS (
                    SELECT 1
                    FROM documents d
                    WHERE d.id = document_chunks.document_id
                      AND d.tenant_id = document_chunks.tenant_id
                      AND d.tenant_id = get_current_tenant_id()
                )
            );
    END IF;
END $$;
