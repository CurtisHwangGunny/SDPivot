-- Restore only rows still in the exact disabled state written by the up migration.
-- State rows are retained as durable evidence; restored_at records successful rollback.

DO $$
BEGIN
    IF to_regclass('public.sdpivot_disable_legacy_ops_admin_000013_state') IS NULL THEN
        RETURN;
    END IF;

    WITH restored AS (
        UPDATE users AS target
        SET password_hash = state.original_password_hash,
            is_active = state.original_is_active,
            must_change_password = state.original_must_change_password,
            is_ops_admin = state.original_is_ops_admin,
            is_system_admin = state.original_is_system_admin,
            updated_at = NOW()
        FROM sdpivot_disable_legacy_ops_admin_000013_state AS state
        WHERE target.id = state.user_id
          AND target.email = state.email
          AND state.disabled_password_hash =
              '!sdpivot-disabled-legacy-ops-admin:' || state.user_id
          AND target.password_hash = state.disabled_password_hash
          AND target.is_active = FALSE
          AND target.must_change_password = TRUE
          AND state.restored_at IS NULL
        RETURNING state.user_id
    )
    UPDATE sdpivot_disable_legacy_ops_admin_000013_state AS state
    SET restored_at = NOW()
    FROM restored
    WHERE state.user_id = restored.user_id
      AND state.restored_at IS NULL;
END $$;
