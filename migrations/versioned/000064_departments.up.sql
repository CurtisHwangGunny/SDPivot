-- Migration: 000064_departments
-- Description: Add tenant-scoped department hierarchy management.
CREATE TABLE IF NOT EXISTS departments (
    id          VARCHAR(36) PRIMARY KEY,
    tenant_id   BIGINT NOT NULL,
    parent_id   VARCHAR(36) NOT NULL DEFAULT '',
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_departments_tenant_parent
    ON departments (tenant_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_departments_deleted_at
    ON departments (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_sibling_name
    ON departments (tenant_id, parent_id, name)
    WHERE deleted_at IS NULL;
