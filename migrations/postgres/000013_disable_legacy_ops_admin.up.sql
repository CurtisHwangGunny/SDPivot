-- Disable only the untouched legacy fixed operations administrator.
-- Accounts that changed password, email or identity are deliberately excluded.

CREATE TABLE IF NOT EXISTS sdpivot_disable_legacy_ops_admin_000013_state (
    user_id                       VARCHAR(36) PRIMARY KEY,
    email                         VARCHAR(255) NOT NULL,
    original_password_hash        VARCHAR(255) NOT NULL,
    original_is_active            BOOLEAN,
    original_must_change_password BOOLEAN,
    original_is_ops_admin         BOOLEAN,
    original_is_system_admin      BOOLEAN,
    disabled_password_hash        VARCHAR(255) NOT NULL,
    disabled_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    restored_at                   TIMESTAMPTZ
);

INSERT INTO sdpivot_disable_legacy_ops_admin_000013_state (
    user_id,
    email,
    original_password_hash,
    original_is_active,
    original_must_change_password,
    original_is_ops_admin,
    original_is_system_admin,
    disabled_password_hash
)
SELECT
    id,
    email,
    password_hash,
    is_active,
    must_change_password,
    is_ops_admin,
    is_system_admin,
    '!sdpivot-disabled-legacy-ops-admin:' || id
FROM users
WHERE email = 'admin@smartknora.com'
  AND password_hash = '$2a$10$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2'
ON CONFLICT (user_id) DO NOTHING;

-- Re-arm a previously restored state row only when the same legacy account still
-- uses the exact known credential and the stored evidence is intact.
UPDATE sdpivot_disable_legacy_ops_admin_000013_state AS state
SET original_password_hash = target.password_hash,
    original_is_active = target.is_active,
    original_must_change_password = target.must_change_password,
    original_is_ops_admin = target.is_ops_admin,
    original_is_system_admin = target.is_system_admin,
    disabled_at = NOW(),
    restored_at = NULL
FROM users AS target
WHERE state.user_id = target.id
  AND state.email = target.email
  AND target.email = 'admin@smartknora.com'
  AND target.password_hash = '$2a$10$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2'
  AND state.original_password_hash = '$2a$10$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2'
  AND state.disabled_password_hash = '!sdpivot-disabled-legacy-ops-admin:' || state.user_id
  AND state.restored_at IS NOT NULL;

UPDATE users AS target
SET password_hash = state.disabled_password_hash,
    is_active = FALSE,
    must_change_password = TRUE,
    updated_at = NOW()
FROM sdpivot_disable_legacy_ops_admin_000013_state AS state
WHERE target.id = state.user_id
  AND target.email = state.email
  AND target.email = 'admin@smartknora.com'
  AND target.password_hash = '$2a$10$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2'
  AND state.original_password_hash = '$2a$10$L4fDCGy48S7wHDmzZqAAf.sql5NnA1.0uEwfbbVzvAsMSV0qyUGS2'
  AND state.disabled_password_hash = '!sdpivot-disabled-legacy-ops-admin:' || state.user_id
  AND state.restored_at IS NULL;
