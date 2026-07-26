DROP INDEX IF EXISTS idx_users_locked_until;

ALTER TABLE users DROP COLUMN locked_until;
ALTER TABLE users DROP COLUMN failed_login_attempts;
ALTER TABLE users DROP COLUMN password_expires_at;
ALTER TABLE users DROP COLUMN password_changed_at;
ALTER TABLE users DROP COLUMN must_change_password;
