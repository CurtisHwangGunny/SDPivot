-- Fix missing RLS on document_versions
ALTER TABLE document_versions ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_versions_tenant_isolation ON document_versions
  USING (tenant_id = current_setting('app.current_tenant_id')::bigint);

-- Fix missing RLS on qa_messages
ALTER TABLE qa_messages ENABLE ROW LEVEL SECURITY;
CREATE POLICY qa_messages_tenant_isolation ON qa_messages
  USING (tenant_id = current_setting('app.current_tenant_id')::bigint);

-- Fix missing RLS on announcements
ALTER TABLE announcements ENABLE ROW LEVEL SECURITY;
CREATE POLICY announcements_tenant_isolation ON announcements
  USING (tenant_id = current_setting('app.current_tenant_id')::bigint);
