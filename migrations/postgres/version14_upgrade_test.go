package postgres

import (
	"context"
	"os"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestVersion14UpgradePreservesBillingData(t *testing.T) {
	migration15, err := os.ReadFile("000015_remove_billing_subscription.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if destructive := regexp.MustCompile(`(?i)\b(drop\s+(table|column)|delete\s+from|truncate)\b`).Find(migration15); destructive != nil {
		t.Fatalf("version 15 upgrade contains destructive operation %q", destructive)
	}

	withContractDatabase(t, func(ctx context.Context, conn *pgx.Conn) {
		fixture := `
			CREATE EXTENSION IF NOT EXISTS pgcrypto;
			CREATE TABLE sdpivot_schema_migrations (version BIGINT NOT NULL, dirty BOOLEAN NOT NULL);
			INSERT INTO sdpivot_schema_migrations VALUES (14, FALSE);

			CREATE TABLE tenants (
				id INTEGER PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				description TEXT,
				api_key VARCHAR(64) NOT NULL,
				status VARCHAR(50),
				business VARCHAR(255) NOT NULL,
				created_at TIMESTAMPTZ,
				updated_at TIMESTAMPTZ,
				deleted_at TIMESTAMPTZ
			);
			CREATE TABLE users (
				id VARCHAR(36) PRIMARY KEY,
				tenant_id INTEGER,
				is_ops_admin BOOLEAN NOT NULL DEFAULT FALSE,
				is_system_admin BOOLEAN NOT NULL DEFAULT FALSE,
				can_access_all_tenants BOOLEAN NOT NULL DEFAULT FALSE,
				is_active BOOLEAN NOT NULL DEFAULT TRUE,
				paid_at TIMESTAMPTZ,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				deleted_at TIMESTAMPTZ
			);
			CREATE TABLE tenant_members (
				id BIGSERIAL PRIMARY KEY,
				user_id VARCHAR(36) NOT NULL,
				tenant_id INTEGER NOT NULL,
				role VARCHAR(20) NOT NULL,
				status VARCHAR(20) NOT NULL,
				joined_at TIMESTAMPTZ NOT NULL,
				created_at TIMESTAMPTZ NOT NULL,
				updated_at TIMESTAMPTZ NOT NULL,
				deleted_at TIMESTAMPTZ
			);
			CREATE UNIQUE INDEX idx_tenant_members_user_tenant_unique
				ON tenant_members(user_id, tenant_id) WHERE deleted_at IS NULL;
			CREATE TABLE org_ext (org_id VARCHAR(36) PRIMARY KEY, subscription_status VARCHAR(20));
			CREATE TABLE system_configs (
				id BIGSERIAL PRIMARY KEY,
				key VARCHAR(100) NOT NULL UNIQUE,
				value TEXT NOT NULL DEFAULT '',
				description VARCHAR(255) NOT NULL DEFAULT '',
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);
			CREATE TABLE billing_plans (id VARCHAR(36) PRIMARY KEY, name VARCHAR(100) NOT NULL);
			CREATE TABLE enterprise_subscriptions (
				id VARCHAR(36) PRIMARY KEY,
				org_id VARCHAR(36) NOT NULL,
				plan_id VARCHAR(36) NOT NULL
			);
			CREATE TABLE invoices (id VARCHAR(36) PRIMARY KEY, org_id VARCHAR(36) NOT NULL);

			INSERT INTO users (id, is_ops_admin, is_system_admin, can_access_all_tenants, paid_at)
			VALUES ('ops-user', TRUE, TRUE, TRUE, NOW());
			INSERT INTO org_ext VALUES ('customer-org', 'paid');
			INSERT INTO system_configs (key, value) VALUES ('downgrade_space_limit', 'retained');
			INSERT INTO billing_plans VALUES ('plan-1', 'Enterprise');
			INSERT INTO enterprise_subscriptions VALUES ('subscription-1', 'customer-org', 'plan-1');
			INSERT INTO invoices VALUES ('invoice-1', 'customer-org');`
		if err := execContractSQL(t, ctx, conn, fixture); err != nil {
			t.Fatalf("create version 14 fixture: %v", err)
		}

		migrations := []struct {
			version int
			name    string
		}{
			{15, "000015_remove_billing_subscription.up.sql"},
			{16, "000016_sms_provider_config.up.sql"},
			{17, "000017_op_admin_auth_rbac.up.sql"},
		}
		for _, migrationFile := range migrations {
			migration, err := os.ReadFile(migrationFile.name)
			if err != nil {
				t.Fatal(err)
			}
			if err := execContractSQL(t, ctx, conn, string(migration)); err != nil {
				t.Fatalf("apply migration %d: %v", migrationFile.version, err)
			}
			if _, err := conn.Exec(ctx, "UPDATE sdpivot_schema_migrations SET version = $1", migrationFile.version); err != nil {
				t.Fatalf("record migration %d: %v", migrationFile.version, err)
			}
		}

		for _, name := range []string{
			"000015_remove_billing_subscription.up.sql",
			"000016_sms_provider_config.up.sql",
			"000017_op_admin_auth_rbac.up.sql",
		} {
			migration, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := execContractSQL(t, ctx, conn, string(migration)); err != nil {
				t.Fatalf("reapply %s: %v", name, err)
			}
		}

		var preserved int
		if err := conn.QueryRow(ctx, `
			SELECT
				(SELECT count(*) FROM billing_plans WHERE id = 'plan-1') +
				(SELECT count(*) FROM enterprise_subscriptions WHERE id = 'subscription-1') +
				(SELECT count(*) FROM invoices WHERE id = 'invoice-1') +
				(SELECT count(*) FROM org_ext WHERE org_id = 'customer-org' AND subscription_status = 'paid') +
				(SELECT count(*) FROM users WHERE id = 'ops-user' AND paid_at IS NOT NULL) +
				(SELECT count(*) FROM system_configs WHERE key = 'downgrade_space_limit' AND value = 'retained')`).Scan(&preserved); err != nil {
			t.Fatalf("inspect preserved version 14 data: %v", err)
		}
		if preserved != 6 {
			t.Fatalf("preserved version 14 records = %d, want 6", preserved)
		}

		var version int
		var normalized, member bool
		if err := conn.QueryRow(ctx, `
			SELECT m.version,
			       u.tenant_id = 1 AND NOT u.can_access_all_tenants AND u.access_role = 'super_admin',
			       EXISTS (
			           SELECT 1 FROM tenant_members tm
			           WHERE tm.user_id = u.id AND tm.tenant_id = 1 AND tm.role = 'owner'
			             AND tm.status = 'active' AND tm.deleted_at IS NULL
			       )
			FROM sdpivot_schema_migrations m
			JOIN users u ON u.id = 'ops-user'`).Scan(&version, &normalized, &member); err != nil {
			t.Fatalf("inspect current upgrade state: %v", err)
		}
		if version != 17 || !normalized || !member {
			t.Fatalf("unexpected current state: version=%d normalized=%t member=%t", version, normalized, member)
		}
	})
}
