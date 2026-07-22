-- Remove legacy permissive tenant policies and replace them with OP-safe isolation.

CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id BIGINT, p_is_ops_admin BOOLEAN DEFAULT FALSE)
RETURNS void
LANGUAGE plpgsql
SECURITY INVOKER
SET search_path = pg_catalog, public
AS $$
BEGIN
    PERFORM pg_catalog.set_config('app.current_tenant_id', p_tenant_id::TEXT, true);
    PERFORM pg_catalog.set_config('app.is_ops_admin', 'false', true);
END;
$$;

CREATE OR REPLACE FUNCTION is_ops_admin_context()
RETURNS BOOLEAN
LANGUAGE sql
STABLE
SECURITY INVOKER
SET search_path = pg_catalog, public
AS $$
    SELECT FALSE;
$$;

DO $$
DECLARE
    target_table TEXT;
    missing_tables TEXT[] := ARRAY[]::TEXT[];
BEGIN
    FOREACH target_table IN ARRAY ARRAY[
        'write_category_config',
        'knowledge_spaces',
        'documents',
        'qa_sessions',
        'qa_messages',
        'writing_drafts',
        'announcements',
        'token_usage',
        'document_chunks'
    ] LOOP
        IF to_regclass('public.' || target_table) IS NULL THEN
            missing_tables := array_append(missing_tables, target_table);
        END IF;
    END LOOP;

    IF cardinality(missing_tables) > 0 THEN
        RAISE EXCEPTION 'SDPivot secure tenant RLS requires target tables: %', array_to_string(missing_tables, ', ');
    END IF;
END $$;

-- Remove every known permissive policy from both historical and Bootstrap paths.
DROP POLICY IF EXISTS wcc_tenant_isolation ON write_category_config;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_write_category_config ON write_category_config;
DROP POLICY IF EXISTS sdpivot_secure_000014_write_category_config ON write_category_config;

DROP POLICY IF EXISTS ks_tenant_isolation ON knowledge_spaces;
DROP POLICY IF EXISTS tenant_isolation ON knowledge_spaces;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_knowledge_spaces ON knowledge_spaces;
DROP POLICY IF EXISTS sdpivot_secure_000014_knowledge_spaces ON knowledge_spaces;

DROP POLICY IF EXISTS doc_tenant_isolation ON documents;
DROP POLICY IF EXISTS tenant_isolation ON documents;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_documents ON documents;
DROP POLICY IF EXISTS sdpivot_secure_000014_documents ON documents;

DROP POLICY IF EXISTS qs_tenant_isolation ON qa_sessions;
DROP POLICY IF EXISTS tenant_isolation ON qa_sessions;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_qa_sessions ON qa_sessions;
DROP POLICY IF EXISTS sdpivot_secure_000014_qa_sessions ON qa_sessions;

DROP POLICY IF EXISTS qm_tenant_isolation ON qa_messages;
DROP POLICY IF EXISTS qa_messages_tenant_isolation ON qa_messages;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_qa_messages ON qa_messages;
DROP POLICY IF EXISTS sdpivot_secure_000014_qa_messages ON qa_messages;

DROP POLICY IF EXISTS wd_tenant_isolation ON writing_drafts;
DROP POLICY IF EXISTS tenant_isolation ON writing_drafts;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_writing_drafts ON writing_drafts;
DROP POLICY IF EXISTS sdpivot_secure_000014_writing_drafts ON writing_drafts;

DROP POLICY IF EXISTS ann_tenant_isolation ON announcements;
DROP POLICY IF EXISTS announcements_tenant_isolation ON announcements;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_announcements ON announcements;
DROP POLICY IF EXISTS sdpivot_secure_000014_announcements ON announcements;

DROP POLICY IF EXISTS tu_tenant_isolation ON token_usage;
DROP POLICY IF EXISTS tenant_isolation ON token_usage;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_token_usage ON token_usage;
DROP POLICY IF EXISTS sdpivot_secure_000014_token_usage ON token_usage;

DROP POLICY IF EXISTS tenant_isolation ON document_chunks;
DROP POLICY IF EXISTS document_chunks_tenant_isolation_000012 ON document_chunks;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000012_document_chunks ON document_chunks;
DROP POLICY IF EXISTS sdpivot_secure_000014_document_chunks ON document_chunks;

ALTER TABLE write_category_config ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_write_category_config ON write_category_config
    FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE knowledge_spaces ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_knowledge_spaces ON knowledge_spaces
    FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_documents ON documents
    FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE qa_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_qa_sessions ON qa_sessions
    FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE qa_messages ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_qa_messages ON qa_messages
    FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE writing_drafts ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_writing_drafts ON writing_drafts
    FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE announcements ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_announcements ON announcements
    FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE token_usage ENABLE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_token_usage ON token_usage
    FOR ALL
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY;
CREATE POLICY sdpivot_secure_000014_document_chunks ON document_chunks
    FOR ALL
    USING (
        EXISTS (
            SELECT 1
              FROM documents d
             WHERE d.id = document_chunks.document_id
               AND d.tenant_id = document_chunks.tenant_id
               AND d.tenant_id = get_current_tenant_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1
              FROM documents d
             WHERE d.id = document_chunks.document_id
               AND d.tenant_id = document_chunks.tenant_id
               AND d.tenant_id = get_current_tenant_id()
        )
    );
