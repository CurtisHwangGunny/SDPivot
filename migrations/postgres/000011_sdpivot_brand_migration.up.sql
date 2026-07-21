-- Migrate only exact, known brand display values and retain a durable rollback map.
-- Historical table names, migration names, user identifiers, documents and paths are untouched.

CREATE TABLE IF NOT EXISTS sdpivot_brand_migration_000011 (
    table_name VARCHAR(63) NOT NULL,
    row_id TEXT NOT NULL,
    column_name VARCHAR(63) NOT NULL,
    old_value TEXT NOT NULL,
    new_value TEXT NOT NULL,
    PRIMARY KEY (table_name, row_id, column_name)
);

DO $$
BEGIN
    IF to_regclass('tenants') IS NOT NULL THEN
        INSERT INTO sdpivot_brand_migration_000011
            (table_name, row_id, column_name, old_value, new_value)
        SELECT 'tenants', target.id::text, 'description', target.description, known.new_value
        FROM tenants AS target
        JOIN (VALUES
            ('SmartKnora', 'SDPivot'),
            ('smartKnora', 'SDPivot'),
            ('随越·智枢', 'SDPivot'),
            ('SMK', 'SDP'),
            ('SmartKnora 默认工作区', 'SDPivot 默认工作区'),
            ('SmartKnora RLS verification tenant A', 'SDPivot RLS verification tenant A'),
            ('SmartKnora RLS verification tenant B', 'SDPivot RLS verification tenant B')
        ) AS known(old_value, new_value) ON target.description = known.old_value
        ON CONFLICT DO NOTHING;

        INSERT INTO sdpivot_brand_migration_000011
            (table_name, row_id, column_name, old_value, new_value)
        SELECT 'tenants', target.id::text, 'business', target.business, known.new_value
        FROM tenants AS target
        JOIN (VALUES
            ('SmartKnora', 'SDPivot'),
            ('smartKnora', 'SDPivot'),
            ('随越·智枢', 'SDPivot'),
            ('SMK', 'SDP'),
            ('smartknora', 'sdpivot'),
            ('smartknora-rls', 'sdpivot-rls')
        ) AS known(old_value, new_value) ON target.business = known.old_value
        ON CONFLICT DO NOTHING;

        UPDATE tenants AS target
        SET description = mapping.new_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'tenants'
          AND mapping.column_name = 'description'
          AND target.id::text = mapping.row_id
          AND target.description = mapping.old_value;

        UPDATE tenants AS target
        SET business = mapping.new_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'tenants'
          AND mapping.column_name = 'business'
          AND target.id::text = mapping.row_id
          AND target.business = mapping.old_value;
    END IF;

    IF to_regclass('organizations') IS NOT NULL THEN
        INSERT INTO sdpivot_brand_migration_000011
            (table_name, row_id, column_name, old_value, new_value)
        SELECT 'organizations', target.id::text, 'description', target.description, known.new_value
        FROM organizations AS target
        JOIN (VALUES
            ('SmartKnora', 'SDPivot'),
            ('smartKnora', 'SDPivot'),
            ('随越·智枢', 'SDPivot'),
            ('SMK', 'SDP'),
            ('SmartKnora 默认工作区', 'SDPivot 默认工作区'),
            ('SmartKnora RLS verification tenant A', 'SDPivot RLS verification tenant A'),
            ('SmartKnora RLS verification tenant B', 'SDPivot RLS verification tenant B')
        ) AS known(old_value, new_value) ON target.description = known.old_value
        ON CONFLICT DO NOTHING;

        UPDATE organizations AS target
        SET description = mapping.new_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'organizations'
          AND mapping.column_name = 'description'
          AND target.id::text = mapping.row_id
          AND target.description = mapping.old_value;
    END IF;

    IF to_regclass('system_configs') IS NOT NULL THEN
        INSERT INTO sdpivot_brand_migration_000011
            (table_name, row_id, column_name, old_value, new_value)
        SELECT 'system_configs', target.id::text, 'value', target.value, known.new_value
        FROM system_configs AS target
        JOIN (VALUES
            ('SmartKnora', 'SDPivot'),
            ('smartKnora', 'SDPivot'),
            ('随越·智枢', 'SDPivot'),
            ('SMK', 'SDP')
        ) AS known(old_value, new_value) ON target.value = known.old_value
        WHERE target.key IN ('product_name', 'application_name', 'brand_name', 'product_short_name')
        ON CONFLICT DO NOTHING;

        UPDATE system_configs AS target
        SET value = mapping.new_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'system_configs'
          AND mapping.column_name = 'value'
          AND target.id::text = mapping.row_id
          AND target.value = mapping.old_value;
    END IF;
END $$;
