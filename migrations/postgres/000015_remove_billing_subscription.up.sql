DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS enterprise_subscriptions;
DROP TABLE IF EXISTS billing_plans;

ALTER TABLE org_ext DROP COLUMN IF EXISTS subscription_status;
ALTER TABLE users DROP COLUMN IF EXISTS paid_at;

DELETE FROM system_configs WHERE key = 'downgrade_space_limit';
