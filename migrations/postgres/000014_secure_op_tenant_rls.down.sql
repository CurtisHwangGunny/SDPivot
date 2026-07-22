-- Roll back policy names without restoring the legacy operations-admin bypass.

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
            CONTINUE;
        END IF;
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'sdpivot_secure_000014_' || target_table, target_table);
    END LOOP;
END $$;

DO $$
BEGIN
    IF to_regclass('public.write_category_config') IS NOT NULL THEN
        DROP POLICY IF EXISTS wcc_tenant_isolation ON write_category_config;
        CREATE POLICY wcc_tenant_isolation ON write_category_config
            FOR ALL USING (tenant_id = get_current_tenant_id())
            WITH CHECK (tenant_id = get_current_tenant_id());
    END IF;

    IF to_regclass('public.knowledge_spaces') IS NOT NULL THEN
        DROP POLICY IF EXISTS ks_tenant_isolation ON knowledge_spaces;
        CREATE POLICY ks_tenant_isolation ON knowledge_spaces
            FOR ALL USING (tenant_id = get_current_tenant_id())
            WITH CHECK (tenant_id = get_current_tenant_id());
    END IF;

    IF to_regclass('public.documents') IS NOT NULL THEN
        DROP POLICY IF EXISTS doc_tenant_isolation ON documents;
        CREATE POLICY doc_tenant_isolation ON documents
            FOR ALL USING (tenant_id = get_current_tenant_id())
            WITH CHECK (tenant_id = get_current_tenant_id());
    END IF;

    IF to_regclass('public.qa_sessions') IS NOT NULL THEN
        DROP POLICY IF EXISTS qs_tenant_isolation ON qa_sessions;
        CREATE POLICY qs_tenant_isolation ON qa_sessions
            FOR ALL USING (tenant_id = get_current_tenant_id())
            WITH CHECK (tenant_id = get_current_tenant_id());
    END IF;

    IF to_regclass('public.qa_messages') IS NOT NULL THEN
        DROP POLICY IF EXISTS qm_tenant_isolation ON qa_messages;
        CREATE POLICY qm_tenant_isolation ON qa_messages
            FOR ALL USING (tenant_id = get_current_tenant_id())
            WITH CHECK (tenant_id = get_current_tenant_id());
    END IF;

    IF to_regclass('public.writing_drafts') IS NOT NULL THEN
        DROP POLICY IF EXISTS wd_tenant_isolation ON writing_drafts;
        CREATE POLICY wd_tenant_isolation ON writing_drafts
            FOR ALL USING (tenant_id = get_current_tenant_id())
            WITH CHECK (tenant_id = get_current_tenant_id());
    END IF;

    IF to_regclass('public.announcements') IS NOT NULL THEN
        DROP POLICY IF EXISTS ann_tenant_isolation ON announcements;
        CREATE POLICY ann_tenant_isolation ON announcements
            FOR ALL USING (tenant_id = get_current_tenant_id())
            WITH CHECK (tenant_id = get_current_tenant_id());
    END IF;

    IF to_regclass('public.token_usage') IS NOT NULL THEN
        DROP POLICY IF EXISTS tu_tenant_isolation ON token_usage;
        CREATE POLICY tu_tenant_isolation ON token_usage
            FOR ALL USING (tenant_id = get_current_tenant_id())
            WITH CHECK (tenant_id = get_current_tenant_id());
    END IF;

    IF to_regclass('public.document_chunks') IS NOT NULL
       AND to_regclass('public.documents') IS NOT NULL THEN
        ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY;
        ALTER TABLE document_chunks FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS document_chunks_tenant_isolation_000012 ON document_chunks;
        CREATE POLICY document_chunks_tenant_isolation_000012 ON document_chunks
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
    END IF;
END $$;
