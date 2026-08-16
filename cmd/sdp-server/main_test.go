package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateStandaloneConfig(t *testing.T) {
	for _, name := range []string{"SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET", "SDP_DB_PASSWORD", "SMART_DB_PASSWORD", "DOCREADER_ADDR"} {
		t.Setenv(name, "")
	}
	product := &config.ProductConfig{OPMode: true}

	t.Setenv("SDP_DB_PASSWORD", "db-password")
	if err := validateStandaloneConfig(product); err == nil || !strings.Contains(err.Error(), "JWT") {
		t.Fatalf("expected missing JWT error, got %v", err)
	}

	t.Setenv("SDP_DB_PASSWORD", "")
	t.Setenv("SDP_JWT_SECRET", "jwt-secret")
	if err := validateStandaloneConfig(product); err == nil || !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("expected missing DB password error, got %v", err)
	}

	t.Setenv("SDP_JWT_SECRET", "")
	t.Setenv("SMARTKNORA_JWT_SECRET", "legacy-jwt")
	t.Setenv("SMART_DB_PASSWORD", "legacy-db")
	if err := validateStandaloneConfig(product); err == nil || !strings.Contains(err.Error(), "DOCREADER_ADDR") {
		t.Fatalf("expected missing docreader error, got %v", err)
	}

	t.Setenv("DOCREADER_ADDR", "docreader:50051")
	if err := validateStandaloneConfig(product); err != nil {
		t.Fatalf("legacy compatibility variables should pass: %v", err)
	}
}

func TestValidateStandaloneConfigNonOPAllowsDefaults(t *testing.T) {
	for _, name := range []string{"SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET", "SDP_DB_PASSWORD", "SMART_DB_PASSWORD", "DOCREADER_ADDR"} {
		t.Setenv(name, "")
	}
	if err := validateStandaloneConfig(&config.ProductConfig{}); err != nil {
		t.Fatalf("non-OP mode should preserve defaults: %v", err)
	}
}

func TestLoadDBPoolConfig(t *testing.T) {
	setPoolEnv := func(open, idle, lifetime string) {
		t.Setenv("SDP_DB_MAX_OPEN_CONNS", open)
		t.Setenv("SDP_DB_MAX_IDLE_CONNS", idle)
		t.Setenv("SDP_DB_CONN_MAX_LIFETIME", lifetime)
	}

	t.Run("defaults", func(t *testing.T) {
		setPoolEnv("", "", "")
		cfg, err := loadDBPoolConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxOpenConns != 10 || cfg.MaxIdleConns != 5 || cfg.ConnMaxLifetime != 10*time.Minute {
			t.Fatalf("unexpected defaults: %+v", cfg)
		}
	})

	t.Run("valid overrides", func(t *testing.T) {
		setPoolEnv("8", "2", "30m")
		cfg, err := loadDBPoolConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxOpenConns != 8 || cfg.MaxIdleConns != 2 || cfg.ConnMaxLifetime != 30*time.Minute {
			t.Fatalf("unexpected overrides: %+v", cfg)
		}
	})

	for _, tc := range []struct {
		name     string
		open     string
		idle     string
		lifetime string
	}{
		{name: "invalid integer", open: "invalid", idle: "2", lifetime: "10m"},
		{name: "idle exceeds open", open: "2", idle: "3", lifetime: "10m"},
		{name: "invalid duration", open: "10", idle: "5", lifetime: "forever"},
		{name: "non-positive duration", open: "10", idle: "5", lifetime: "0s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setPoolEnv(tc.open, tc.idle, tc.lifetime)
			if _, err := loadDBPoolConfig(); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestLoadReadyTimeout(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("SDP_READY_TIMEOUT", "")
		timeout, err := loadReadyTimeout()
		if err != nil || timeout != 3*time.Second {
			t.Fatalf("timeout=%v err=%v", timeout, err)
		}
	})
	t.Run("override", func(t *testing.T) {
		t.Setenv("SDP_READY_TIMEOUT", "750ms")
		timeout, err := loadReadyTimeout()
		if err != nil || timeout != 750*time.Millisecond {
			t.Fatalf("timeout=%v err=%v", timeout, err)
		}
	})
	for _, value := range []string{"invalid", "0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("SDP_READY_TIMEOUT", value)
			if _, err := loadReadyTimeout(); err == nil {
				t.Fatal("expected readiness timeout error")
			}
		})
	}
}

func TestTopLevelHealthAndReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	internalError := "password=super-secret internal connection failure"
	r := gin.New()
	registerTopLevelHealth(r, time.Second, func(context.Context) error {
		return errors.New(internalError)
	})

	health := httptest.NewRecorder()
	r.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}

	ready := httptest.NewRecorder()
	r.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", ready.Code)
	}
	if strings.Contains(ready.Body.String(), internalError) || strings.Contains(ready.Body.String(), "super-secret") {
		t.Fatalf("readiness response leaked internal error: %s", ready.Body.String())
	}
}

func TestTopLevelReadinessUsesTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerTopLevelHealth(r, 20*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", response.Code)
	}
}

func TestReadinessDependencyOrderAndShortCircuit(t *testing.T) {
	var calls []string
	check := newReadinessCheck(
		func(context.Context) error {
			calls = append(calls, "db")
			return nil
		},
		func(context.Context) error {
			calls = append(calls, "redis")
			return nil
		},
	)
	if err := check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "db,redis" {
		t.Fatalf("unexpected readiness order: %v", calls)
	}

	calls = nil
	check = newReadinessCheck(
		func(context.Context) error {
			calls = append(calls, "db")
			return errors.New("database unavailable")
		},
		func(context.Context) error {
			calls = append(calls, "redis")
			return nil
		},
	)
	if err := check(context.Background()); err == nil {
		t.Fatal("expected database readiness error")
	}
	if strings.Join(calls, ",") != "db" {
		t.Fatalf("redis should not be called after database failure: %v", calls)
	}
}

func TestRedisReadinessFailureIsSanitized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	internalError := "redis password=super-secret unavailable"
	r := gin.New()
	registerTopLevelHealth(r, time.Second, newReadinessCheck(
		func(context.Context) error { return nil },
		func(context.Context) error { return errors.New(internalError) },
	))

	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), internalError) || strings.Contains(response.Body.String(), "super-secret") {
		t.Fatalf("readiness response leaked redis error: %s", response.Body.String())
	}
}

type closeRecorder struct {
	calls *[]string
	name  string
}

func (c closeRecorder) Close() error {
	*c.calls = append(*c.calls, c.name)
	return nil
}

func TestCloseDependenciesClosesRedisAndDatabase(t *testing.T) {
	var calls []string
	closeDependencies(closeRecorder{calls: &calls, name: "db"}, func() error {
		calls = append(calls, "redis")
		return nil
	})
	if strings.Join(calls, ",") != "redis,db" {
		t.Fatalf("unexpected cleanup calls: %v", calls)
	}
}

func TestTopLevelReadinessSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerTopLevelHealth(r, time.Second, func(context.Context) error { return nil })

	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("readiness status = %d", response.Code)
	}
}

func TestEnsureSDPivotSchemaMigratesRBACAndHandlerModels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, ensureSDPivotSchema(db))

	migrator := db.Migrator()
	require.True(t, migrator.HasColumn(&types.User{}, "access_role"))
	require.True(t, migrator.HasColumn(&types.User{}, "department_id"))
	for _, model := range []interface{}{
		&types.Department{},
		&types.SDPivotUserProfile{},
		&types.KnowledgeSpace{},
		&types.SDPivotDocument{},
		&types.SDPivotDocumentChunk{},
		&types.QASession{},
		&types.WritingDraft{},
		&types.TagDimension{},
		&types.TagDictionary{},
	} {
		require.Truef(t, migrator.HasTable(model), "expected table for %T", model)
	}
}

func TestInitializeOPAdminFromEnvCreatesAndRepairsIdempotently(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.Tenant{}, &types.User{}, &types.TenantMember{}, &types.SDPivotUserProfile{},
		&types.Organization{}, &types.OrganizationTenantMember{}, &types.OrgExt{},
		&types.SDPivotOrgMember{}, &types.KnowledgeSpace{},
	))

	t.Setenv(opAdminEmailEnv, "sysadmin@sdpivot.local")
	t.Setenv(opAdminPasswordEnv, "ValidPass!123")
	require.NoError(t, initializeOPAdminFromEnv(db))

	var user types.User
	require.NoError(t, db.Where("email = ?", "sysadmin@sdpivot.local").First(&user).Error)
	require.Equal(t, types.DefaultTenantID, user.TenantID)
	require.True(t, user.IsOpsAdmin)
	require.True(t, user.IsSystemAdmin)
	require.False(t, user.CanAccessAllTenants)
	require.Equal(t, types.AccessRoleSuperAdmin, user.AccessRole)
	require.True(t, user.MustChangePassword)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("ValidPass!123")))
	originalHash := user.PasswordHash

	require.NoError(t, initializeOPAdminFromEnv(db))
	require.NoError(t, db.Where("email = ?", "sysadmin@sdpivot.local").First(&user).Error)
	require.Equal(t, originalHash, user.PasswordHash, "matching passwords must not be rehashed on every restart")

	var tenantCount, userCount, membershipCount, orgCount, orgTenantMemberCount, orgMemberCount int64
	require.NoError(t, db.Model(&types.Tenant{}).Where("id = ?", types.DefaultTenantID).Count(&tenantCount).Error)
	require.NoError(t, db.Model(&types.User{}).Where("email = ?", user.Email).Count(&userCount).Error)
	require.NoError(t, db.Model(&types.TenantMember{}).Where("user_id = ? AND tenant_id = ? AND role = ? AND status = ?", user.ID, types.DefaultTenantID, types.TenantRoleOwner, types.TenantMemberStatusActive).Count(&membershipCount).Error)
	require.NoError(t, db.Model(&types.Organization{}).Where("id = ? AND owner_tenant_id = ?", types.DefaultOrganizationID, types.DefaultTenantID).Count(&orgCount).Error)
	require.NoError(t, db.Model(&types.OrganizationTenantMember{}).Where("organization_id = ? AND tenant_id = ? AND role = ?", types.DefaultOrganizationID, types.DefaultTenantID, types.OrgRoleAdmin).Count(&orgTenantMemberCount).Error)
	require.NoError(t, db.Model(&types.SDPivotOrgMember{}).Where("org_id = ? AND user_id = ? AND status = ?", types.DefaultOrganizationID, user.ID, "active").Count(&orgMemberCount).Error)
	require.EqualValues(t, 1, tenantCount)
	require.EqualValues(t, 1, userCount)
	require.EqualValues(t, 1, membershipCount)
	require.EqualValues(t, 1, orgCount)
	require.EqualValues(t, 1, orgTenantMemberCount)
	require.EqualValues(t, 1, orgMemberCount)

	var listedOrgCount int64
	require.NoError(t, db.Model(&types.Organization{}).
		Joins("JOIN organization_tenant_members otm ON otm.organization_id = organizations.id").
		Where("otm.tenant_id = ?", types.DefaultTenantID).Count(&listedOrgCount).Error)
	require.EqualValues(t, 1, listedOrgCount)
}

func TestInitializeOPDefaultsWithoutBootstrapAccount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.Tenant{}, &types.User{}, &types.TenantMember{}, &types.SDPivotUserProfile{},
		&types.Organization{}, &types.OrganizationTenantMember{}, &types.OrgExt{},
		&types.SDPivotOrgMember{}, &types.KnowledgeSpace{},
	))
	t.Setenv(opAdminEmailEnv, "")
	t.Setenv(opAdminPasswordEnv, "")
	now := time.Now()
	account := types.User{
		ID: "existing-account", Username: "existing", Email: "existing@sdpivot.local",
		PasswordHash: "unused", TenantID: types.DefaultTenantID, IsActive: true,
		CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, db.Create(&account).Error)
	space := types.KnowledgeSpace{
		ID: "existing-space", TenantID: types.DefaultTenantID, Name: "existing",
		CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, db.Create(&space).Error)

	require.NoError(t, initializeOPAdminFromEnv(db))
	require.NoError(t, initializeOPAdminFromEnv(db))

	var tenantCount, orgCount, membershipCount int64
	require.NoError(t, db.Model(&types.Tenant{}).Where("id = ?", types.DefaultTenantID).Count(&tenantCount).Error)
	require.NoError(t, db.Model(&types.Organization{}).Where("id = ? AND owner_tenant_id = ?", types.DefaultOrganizationID, types.DefaultTenantID).Count(&orgCount).Error)
	require.NoError(t, db.Model(&types.OrganizationTenantMember{}).
		Where("organization_id = ? AND tenant_id = ?", types.DefaultOrganizationID, types.DefaultTenantID).
		Count(&membershipCount).Error)
	require.EqualValues(t, 1, tenantCount)
	require.EqualValues(t, 1, orgCount)
	require.EqualValues(t, 1, membershipCount)

	var org types.Organization
	require.NoError(t, db.First(&org, "id = ?", types.DefaultOrganizationID).Error)
	require.Equal(t, account.ID, org.OwnerID)
	var accountMembershipCount int64
	require.NoError(t, db.Model(&types.SDPivotOrgMember{}).
		Where("org_id = ? AND user_id = ? AND role = ? AND status = ?", types.DefaultOrganizationID, account.ID, "owner", "active").
		Count(&accountMembershipCount).Error)
	require.EqualValues(t, 1, accountMembershipCount)
	require.NoError(t, db.First(&space, "id = ?", space.ID).Error)
	require.Equal(t, types.DefaultOrganizationID, requireStringValue(t, space.OrgID))
}

func requireStringValue(t *testing.T, value *string) string {
	t.Helper()
	require.NotNil(t, value)
	return *value
}

func TestValidateBootstrapAdminPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "valid", password: "ValidPass!123"},
		{name: "too short", password: "Aa1!aaa", wantErr: true},
		{name: "missing uppercase", password: "validpass!123", wantErr: true},
		{name: "missing lowercase", password: "VALIDPASS!123", wantErr: true},
		{name: "missing digit", password: "ValidPassword!", wantErr: true},
		{name: "missing special", password: "ValidPass123", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBootstrapAdminPassword(tt.password)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestInitializeOPAdminFromEnvRepairsExistingPasswordAndTenant(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.Tenant{}, &types.User{}, &types.TenantMember{}, &types.SDPivotUserProfile{},
		&types.Organization{}, &types.OrganizationTenantMember{}, &types.OrgExt{},
		&types.SDPivotOrgMember{}, &types.KnowledgeSpace{},
	))

	oldHash, err := bcrypt.GenerateFromPassword([]byte("OldPass!123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	user := types.User{ID: "existing-admin", Username: "sysadmin", Email: "sysadmin@sdpivot.local", PasswordHash: string(oldHash), TenantID: 99, IsActive: false, CanAccessAllTenants: true}
	require.NoError(t, db.Create(&user).Error)

	t.Setenv(opAdminEmailEnv, user.Email)
	t.Setenv(opAdminPasswordEnv, "NewPass!123")
	require.NoError(t, initializeOPAdminFromEnv(db))
	require.NoError(t, db.Where("id = ?", user.ID).First(&user).Error)
	require.Equal(t, types.DefaultTenantID, user.TenantID)
	require.True(t, user.IsActive && user.IsOpsAdmin && user.IsSystemAdmin)
	require.False(t, user.CanAccessAllTenants)
	require.Equal(t, types.AccessRoleSuperAdmin, user.AccessRole)
	require.True(t, user.MustChangePassword)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("NewPass!123")))
}
