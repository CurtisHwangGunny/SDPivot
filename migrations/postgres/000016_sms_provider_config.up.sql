-- Phase 1.7: configurable SMS provider abstraction.
CREATE TABLE IF NOT EXISTS system_configs (
    id          BIGSERIAL PRIMARY KEY,
    key         VARCHAR(100) NOT NULL UNIQUE,
    value       TEXT NOT NULL DEFAULT '',
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO system_configs (key, value, description) VALUES
    ('sms_provider', 'custom', '短信服务商: aliyun/tencent/huawei/custom'),
    ('sms_endpoint', '', '短信服务 HTTP 接口地址'),
    ('sms_access_key_id', '', '短信服务 Access Key ID'),
    ('sms_access_key_secret', '', '短信服务 Access Key Secret'),
    ('sms_region', '', '短信服务区域'),
    ('sms_sign_name', '', '短信签名'),
    ('sms_template_id', '', '短信模板 ID'),
    ('sms_app_id', '', '短信应用 ID'),
    ('sms_sender', '', '华为短信发送通道号'),
    ('sms_custom_headers', '{}', '自定义短信接口请求头 JSON'),
    ('sms_timeout_seconds', '10', '短信请求超时秒数')
ON CONFLICT (key) DO NOTHING;
