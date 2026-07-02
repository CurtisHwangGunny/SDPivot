-- Rollback: 000001 smartknora init

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS space_members;
DROP TABLE IF EXISTS space_categories;
DROP TABLE IF EXISTS knowledge_spaces;
DROP TABLE IF EXISTS token_usage;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
