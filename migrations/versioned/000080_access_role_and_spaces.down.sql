DROP TRIGGER IF EXISTS users_last_super_admin_guard ON users;
DROP FUNCTION IF EXISTS prevent_last_super_admin_change();
DROP FUNCTION IF EXISTS get_current_access_role();
DROP TABLE IF EXISTS space_members;
DROP TABLE IF EXISTS knowledge_spaces;
DROP INDEX IF EXISTS idx_users_department_id;
DROP INDEX IF EXISTS idx_users_access_role;
ALTER TABLE users DROP COLUMN IF EXISTS department_id;
ALTER TABLE users DROP COLUMN IF EXISTS access_role;
