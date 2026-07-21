DROP POLICY IF EXISTS document_chunks_tenant_isolation_000012 ON document_chunks;

DO $$
DECLARE
    previous_state RECORD;
BEGIN
    SELECT rls_was_enabled, force_was_enabled
    INTO previous_state
    FROM sdpivot_rls_migration_000012_state
    WHERE table_name = 'document_chunks';

    IF FOUND THEN
        IF previous_state.force_was_enabled THEN
            ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY;
        ELSE
            ALTER TABLE document_chunks NO FORCE ROW LEVEL SECURITY;
        END IF;

        IF previous_state.rls_was_enabled THEN
            ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY;
        ELSE
            ALTER TABLE document_chunks DISABLE ROW LEVEL SECURITY;
        END IF;
    END IF;
END $$;

DROP TABLE IF EXISTS sdpivot_rls_migration_000012_state;
