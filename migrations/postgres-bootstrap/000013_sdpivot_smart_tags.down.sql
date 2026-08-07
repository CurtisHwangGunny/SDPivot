DROP POLICY IF EXISTS sdpivot_op_bootstrap_000013_tag_feedback_queue ON tag_feedback_queue;
DROP TABLE IF EXISTS tag_feedback_queue;
DROP POLICY IF EXISTS sdpivot_op_bootstrap_000013_tag_adjustment_log ON tag_adjustment_log;
DROP TABLE IF EXISTS tag_adjustment_log;

DROP POLICY IF EXISTS sdpivot_op_bootstrap_000013_document_tags ON document_tags;
DROP INDEX IF EXISTS idx_document_tags_document_tag;
DROP INDEX IF EXISTS idx_document_tags_id;
ALTER TABLE document_tags DROP CONSTRAINT IF EXISTS document_tags_source_check;
DELETE FROM document_tags AS dt
WHERE NOT EXISTS (SELECT 1 FROM knowledges AS knowledge WHERE knowledge.id = dt.document_id);
ALTER TABLE document_tags DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE document_tags DROP COLUMN IF EXISTS updated_at;
ALTER TABLE document_tags DROP COLUMN IF EXISTS source;
ALTER TABLE document_tags DROP COLUMN IF EXISTS id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_tags_dimension
    ON document_tags (tenant_id, document_id, dimension_id);
ALTER TABLE document_tags ADD CONSTRAINT document_tags_document_id_fkey
    FOREIGN KEY (document_id) REFERENCES knowledges(id) ON DELETE CASCADE;
ALTER TABLE document_tags DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS sdpivot_op_bootstrap_000013_tag_dimensions ON tag_dimensions;
DROP INDEX IF EXISTS idx_tag_dimensions_tenant_enabled;
DROP INDEX IF EXISTS idx_tag_dimensions_tenant_code;
DROP INDEX IF EXISTS idx_tag_dimensions_global_code;
ALTER TABLE tag_dimensions DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE tag_dimensions DROP COLUMN IF EXISTS enabled;
ALTER TABLE tag_dimensions DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE tag_dimensions ADD CONSTRAINT tag_dimensions_code_key UNIQUE (code);
ALTER TABLE tag_dimensions DISABLE ROW LEVEL SECURITY;
