-- 000009_ops_modules.up.sql
-- 运营管理端四模块：敏感词过滤 + 计费管理 + 系统配置 + 模型管理

-- ============================================================
-- 1. sensitive_words 敏感词库
-- ============================================================
CREATE TABLE IF NOT EXISTS sensitive_words (
    id          VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    word        VARCHAR(200) NOT NULL,
    category    VARCHAR(50)  NOT NULL DEFAULT 'general',
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_by  VARCHAR(36),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sw_word ON sensitive_words(word);
CREATE INDEX IF NOT EXISTS idx_sw_category ON sensitive_words(category);
CREATE INDEX IF NOT EXISTS idx_sw_status ON sensitive_words(status);

-- ============================================================
-- 2. billing_plans 计费方案
-- ============================================================
CREATE TABLE IF NOT EXISTS billing_plans (
    id            VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name          VARCHAR(100) NOT NULL,
    price         DECIMAL(10,2) NOT NULL DEFAULT 0,
    token_quota   BIGINT       NOT NULL DEFAULT 0,
    storage_quota BIGINT       NOT NULL DEFAULT 0,
    features      JSONB        NOT NULL DEFAULT '{}',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bp_status ON billing_plans(status);

-- ============================================================
-- 3. enterprise_subscriptions 企业订阅
-- ============================================================
CREATE TABLE IF NOT EXISTS enterprise_subscriptions (
    id         VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    org_id     VARCHAR(36)  NOT NULL,
    plan_id    VARCHAR(36)  NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_es_org ON enterprise_subscriptions(org_id);
CREATE INDEX IF NOT EXISTS idx_es_plan ON enterprise_subscriptions(plan_id);

-- ============================================================
-- 4. invoices 账单
-- ============================================================
CREATE TABLE IF NOT EXISTS invoices (
    id            VARCHAR(36)  PRIMARY KEY DEFAULT gen_random_uuid()::text,
    org_id        VARCHAR(36)  NOT NULL,
    plan_id       VARCHAR(36),
    amount        DECIMAL(10,2) NOT NULL DEFAULT 0,
    period_start  TIMESTAMPTZ  NOT NULL,
    period_end    TIMESTAMPTZ  NOT NULL,
    status        VARCHAR(20)  NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inv_org ON invoices(org_id);
CREATE INDEX IF NOT EXISTS idx_inv_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_inv_period ON invoices(period_start, period_end);
