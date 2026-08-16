DELETE FROM security_settings WHERE section = 'login_lock';

ALTER TABLE security_settings
    DROP CONSTRAINT IF EXISTS security_settings_section_check;

ALTER TABLE security_settings
    ADD CONSTRAINT security_settings_section_check
    CHECK (section IN ('ip_whitelist', 'password_policy', 'session', 'desensitize', 'audit'));
