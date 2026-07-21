-- Restore only rows recorded by 000011 and only while they still hold the migrated value.
-- SDPivot values created after the migration are not present in the map and remain untouched.

DO $$
BEGIN
    IF to_regclass('sdpivot_brand_migration_000011') IS NULL THEN
        RETURN;
    END IF;

    IF to_regclass('tenants') IS NOT NULL THEN
        UPDATE tenants AS target
        SET description = mapping.old_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'tenants'
          AND mapping.column_name = 'description'
          AND target.id::text = mapping.row_id
          AND target.description = mapping.new_value;

        UPDATE tenants AS target
        SET business = mapping.old_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'tenants'
          AND mapping.column_name = 'business'
          AND target.id::text = mapping.row_id
          AND target.business = mapping.new_value;
    END IF;

    IF to_regclass('organizations') IS NOT NULL THEN
        UPDATE organizations AS target
        SET description = mapping.old_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'organizations'
          AND mapping.column_name = 'description'
          AND target.id::text = mapping.row_id
          AND target.description = mapping.new_value;
    END IF;

    IF to_regclass('system_configs') IS NOT NULL THEN
        UPDATE system_configs AS target
        SET value = mapping.old_value
        FROM sdpivot_brand_migration_000011 AS mapping
        WHERE mapping.table_name = 'system_configs'
          AND mapping.column_name = 'value'
          AND target.id::text = mapping.row_id
          AND target.value = mapping.new_value;
    END IF;
END $$;

DROP TABLE IF EXISTS sdpivot_brand_migration_000011;
