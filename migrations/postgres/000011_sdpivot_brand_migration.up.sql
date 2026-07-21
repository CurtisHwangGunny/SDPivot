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
    IF to_regclass('public.tenants') IS NOT NULL THEN
        INSERT INTO sdpivot_brand_migration_000011
            (table_name, row_id, column_name, old_value, new_value)
        SELECT 'tenants', id::text, 'description', description,
               CASE WHEN description = 'SMK' THEN 'SDP' ELSE 'SDPivot' END
        FROM tenants
        WHERE description IN ('SmartKnora', 'smartKnora', '随越·智枢', 'SMK')
        ON CONFLICT DO NOTHING;

        INSERT INTO sdpivot_brand_migration_000011
            (table_name, row_id, column_name, old_value, new_value)
        SELECT 'tenants', id::text, 'business', business,
               CASE WHEN business = 'SMK' THEN 'SDP' ELSE 'SDPivot' END
        FROM tenants
        WHERE business IN ('SmartKnora', 'smartKnora', '随越·智枢', 'SMK')
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

    IF to_regclass('public.organizations') IS NOT NULL THEN
        INSERT INTO sdpivot_brand_migration_000011
            (table_name, row_id, column_name, old_value, new_value)
        SELECT 'organizations', id::text, 'description', description,
               CASE WHEN description = 'SMK' THEN 'SDP' ELSE 'SDPivot' END
        FROM organizations
        WHERE description IN ('SmartKnora', 'smartKnora', '随越·智枢', 'SMK')
        ON CONFLICT DO NOTHING;

        UPDATE organizations AS target
        SET description = mapping.new_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'organizations'
          AND mapping.column_name = 'description'
          AND target.id::text = mapping.row_id
          AND target.description = mapping.old_value;
    END IF;

    IF to_regclass('public.system_configs') IS NOT NULL THEN
        INSERT INTO sdpivot_brand_migration_000011
            (table_name, row_id, column_name, old_value, new_value)
        SELECT 'system_configs', id::text, 'value', value,
               CASE WHEN value = 'SMK' THEN 'SDP' ELSE 'SDPivot' END
        FROM system_configs
        WHERE key IN ('product_name', 'application_name', 'brand_name', 'product_short_name')
          AND value IN ('SmartKnora', 'smartKnora', '随越·智枢', 'SMK')
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
