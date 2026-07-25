-- Migration: 000066_tag_system
-- Description: Add the seven platform tag dimensions, dictionary entries, and document tag assignments.
CREATE TABLE IF NOT EXISTS tag_dimensions (
    id          VARCHAR(36) PRIMARY KEY,
    code        VARCHAR(64) NOT NULL UNIQUE,
    name        VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO tag_dimensions (id, code, name, description, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000001', 'topic', 'Topic', 'Subject or theme of the document', 10),
    ('00000000-0000-0000-0000-000000000002', 'department', 'Department', 'Owning organizational department', 20),
    ('00000000-0000-0000-0000-000000000003', 'business', 'Business', 'Business domain or product line', 30),
    ('00000000-0000-0000-0000-000000000004', 'project', 'Project', 'Related project or initiative', 40),
    ('00000000-0000-0000-0000-000000000005', 'document_type', 'Document Type', 'Document format or functional type', 50),
    ('00000000-0000-0000-0000-000000000006', 'security_level', 'Security Level', 'Information sensitivity classification', 60),
    ('00000000-0000-0000-0000-000000000007', 'lifecycle', 'Lifecycle', 'Document lifecycle or retention stage', 70)
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS tag_dictionary (
    id           VARCHAR(36) PRIMARY KEY,
    dimension_id VARCHAR(36) NOT NULL REFERENCES tag_dimensions(id) ON DELETE RESTRICT,
    name         VARCHAR(128) NOT NULL,
    color        VARCHAR(32) NOT NULL DEFAULT '',
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_dictionary_dimension_name
    ON tag_dictionary (dimension_id, LOWER(name));
CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_dictionary_id_dimension
    ON tag_dictionary (id, dimension_id);
CREATE INDEX IF NOT EXISTS idx_tag_dictionary_dimension_sort
    ON tag_dictionary (dimension_id, sort_order, name);

-- Preserve the Phase 2 JSON dictionary by assigning legacy entries to Topic.
INSERT INTO tag_dictionary (id, dimension_id, name, color, sort_order, created_at, updated_at)
SELECT
    COALESCE(NULLIF(tag ->> 'id', ''), uuid_generate_v4()::text),
    '00000000-0000-0000-0000-000000000001',
    BTRIM(tag ->> 'name'),
    COALESCE(BTRIM(tag ->> 'color'), ''),
    COALESCE((tag ->> 'sort_order')::INTEGER, 0),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM system_configs,
     LATERAL jsonb_array_elements(value::jsonb -> 'tags') AS tag
WHERE key = 'tag_dictionary'
  AND jsonb_typeof(value::jsonb -> 'tags') = 'array'
  AND BTRIM(COALESCE(tag ->> 'name', '')) <> ''
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS document_tags (
    tenant_id    BIGINT NOT NULL,
    document_id  VARCHAR(36) NOT NULL REFERENCES knowledges(id) ON DELETE CASCADE,
    tag_id       VARCHAR(36) NOT NULL,
    dimension_id VARCHAR(36) NOT NULL REFERENCES tag_dimensions(id) ON DELETE RESTRICT,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, document_id, tag_id),
    FOREIGN KEY (tag_id, dimension_id) REFERENCES tag_dictionary(id, dimension_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_document_tags_document
    ON document_tags (tenant_id, document_id);
CREATE INDEX IF NOT EXISTS idx_document_tags_tag
    ON document_tags (tenant_id, tag_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_tags_dimension
    ON document_tags (tenant_id, document_id, dimension_id);
