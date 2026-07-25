DELETE FROM system_configs WHERE key IN (
    'sms_provider', 'sms_endpoint', 'sms_access_key_id', 'sms_access_key_secret',
    'sms_region', 'sms_sign_name', 'sms_template_id', 'sms_app_id', 'sms_sender',
    'sms_custom_headers', 'sms_timeout_seconds'
);
