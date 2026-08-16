-- Conservative rollback for the SDPivot OP bootstrap baseline.
-- Shared tables, columns, business data, RLS enablement, and historical helper
-- functions are retained because the application can depend on them.

DO $$
DECLARE
    target RECORD;
BEGIN
    FOR target IN
        SELECT * FROM (VALUES
            ('organizations', 'sdpivot_op_bootstrap_000012_organizations'),
            ('org_ext', 'sdpivot_op_bootstrap_000012_org_ext'),
            ('org_members', 'sdpivot_op_bootstrap_000012_org_members'),
            ('smartknora_user_profiles', 'sdpivot_op_bootstrap_000012_user_profiles'),
            ('refresh_tokens', 'sdpivot_op_bootstrap_000012_refresh_tokens'),
            ('token_usage', 'sdpivot_op_bootstrap_000012_token_usage'),
            ('knowledge_spaces', 'sdpivot_op_bootstrap_000012_knowledge_spaces'),
            ('space_members', 'sdpivot_op_bootstrap_000012_space_members'),
            ('space_categories', 'sdpivot_op_bootstrap_000012_space_categories'),
            ('documents', 'sdpivot_op_bootstrap_000012_documents'),
            ('document_chunks', 'sdpivot_op_bootstrap_000012_document_chunks'),
            ('document_versions', 'sdpivot_op_bootstrap_000012_document_versions'),
            ('chunk_strategies', 'sdpivot_op_bootstrap_000012_chunk_strategies'),
            ('qa_sessions', 'sdpivot_op_bootstrap_000012_qa_sessions'),
            ('qa_messages', 'sdpivot_op_bootstrap_000012_qa_messages'),
            ('writing_drafts', 'sdpivot_op_bootstrap_000012_writing_drafts'),
            ('write_category_config', 'sdpivot_op_bootstrap_000012_write_category_config'),
            ('announcements', 'sdpivot_op_bootstrap_000012_announcements'),
            ('audit_logs', 'sdpivot_op_bootstrap_000012_audit_logs')
        ) AS policies(table_name, policy_name)
    LOOP
        IF to_regclass('public.' || target.table_name) IS NOT NULL THEN
            EXECUTE format('DROP POLICY IF EXISTS %I ON %I', target.policy_name, target.table_name);
        END IF;
    END LOOP;
END $$;
