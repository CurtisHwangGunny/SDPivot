// SDPivot standalone server
// Connects directly to PostgreSQL + Redis, serves SDPivot API
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/router"
	"github.com/Tencent/WeKnora/internal/types"
)

func main() {
	// ── Config from env ────────────────────────────────────────
	dbHost := getEnvFallback("SDP_DB_HOST", "SMART_DB_HOST", "localhost")
	dbPort := getEnvFallback("SDP_DB_PORT", "SMART_DB_PORT", "5432")
	dbUser := getEnvFallback("SDP_DB_USER", "SMART_DB_USER", "postgres")
	dbPass := getEnvFallback("SDP_DB_PASSWORD", "SMART_DB_PASSWORD", "postgres")
	dbName := getEnvFallback("SDP_DB_NAME", "SMART_DB_NAME", "WeKnora")
	redisAddr := getEnvFallback("SDP_REDIS_ADDR", "REDIS_ADDR", "")
	redisPassword := getEnvFallback("SDP_REDIS_PASSWORD", "REDIS_PASSWORD", "")
	port := getEnv("PORT", "8081")
	product := config.DefaultProductConfig()
	if err := config.ApplyProductEnvOverrides(product); err != nil {
		log.Fatalf("Invalid product configuration: %v", err)
	}
	if err := validateStandaloneConfig(product); err != nil {
		log.Fatalf("Invalid standalone configuration: %v", err)
	}
	poolConfig, err := loadDBPoolConfig()
	if err != nil {
		log.Fatalf("Invalid database pool configuration: %v", err)
	}
	readyTimeout, err := loadReadyTimeout()
	if err != nil {
		log.Fatalf("Invalid readiness configuration: %v", err)
	}

	// ── Database ───────────────────────────────────────────────
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		dbHost, dbPort, dbUser, dbPass, dbName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	var redisClient *redis.Client
	defer closeDependencies(sqlDB, func() error {
		if redisClient == nil {
			return nil
		}
		return redisClient.Close()
	})
	log.Printf("[DB] Connected to PostgreSQL %s:%s/%s", dbHost, dbPort, dbName)
	if err := initializeOPAdminFromEnv(db); err != nil {
		log.Fatalf("Failed to initialize OP administrator: %v", err)
	}

	// ── Redis (optional) ───────────────────────────────────────
	if redisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: redisAddr, Password: redisPassword})
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			log.Printf("[Redis] Warning: %v", err)
		} else {
			log.Printf("[Redis] Connected to %s", redisAddr)
		}
	}

	// ── Gin Router ─────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     getAllowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Health checks ──────────────────────────────────────────
	registerTopLevelHealth(r, readyTimeout, newReadinessCheck(
		sqlDB.PingContext,
		func(ctx context.Context) error {
			if redisClient == nil {
				return nil
			}
			return redisClient.Ping(ctx).Err()
		},
	))

	router.NewSDPivotRouter(router.SDPivotRouterParams{
		DB:          db,
		RedisClient: redisClient,
		Config:      &config.Config{Product: product},
	}).RegisterRoutes(r)

	// ── Start Server ───────────────────────────────────────────
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	serveErr := make(chan error, 1)
	go func() {
		log.Printf("[Server] SDPivot v2.0.0 starting on :%s", port)
		serveErr <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		log.Println("[Server] Shutdown signal received")
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[Server] Listen failed: %v", err)
		}
	}

	log.Println("[Server] Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[Server] Shutdown warning: %v", err)
	}
	log.Println("[Server] Stopped")
}

const (
	opAdminEmailEnv    = "SDP_BOOTSTRAP_ADMIN_EMAIL"
	opAdminPasswordEnv = "SDP_BOOTSTRAP_ADMIN_PASSWORD"
)
	opSysAdminEmailEnv    = "SDP_BOOTSTRAP_SYSADMIN_EMAIL"
	opSysAdminPasswordEnv = "SDP_BOOTSTRAP_SYSADMIN_PASSWORD"

func initializeOPAdminFromEnv(db *gorm.DB) error {
	email := strings.TrimSpace(os.Getenv(opAdminEmailEnv))
	password := os.Getenv(opAdminPasswordEnv)
{} == "" {
		if email != "" || password != "" {
			return fmt.Errorf("%s and %s must be configured together", opAdminEmailEnv, opAdminPasswordEnv)
		}
	}
	if password != "" && len(password) < 8 {
		return fmt.Errorf("%s must be at least 8 characters", opAdminPasswordEnv)
	}

	now := time.Now()
	return db.Transaction(func(tx *gorm.DB) error {
		tenant := types.Tenant{
			ID:          types.DefaultTenantID,
			Name:        "SDPivot",
			Description: "SDPivot OP tenant",
			APIKey:      uuid.NewString(),
			Status:      "active",
			Business:    "sdpivot",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Unscoped().Where("id = ?", types.DefaultTenantID).FirstOrCreate(&tenant).Error; err != nil {
			return fmt.Errorf("ensure OP tenant: %w", err)
		}
		if err := tx.Unscoped().Model(&tenant).Updates(map[string]interface{}{
			"status": "active", "deleted_at": nil, "updated_at": now,
		}).Error; err != nil {
			return fmt.Errorf("repair OP tenant: %w", err)
		}

		var ownerID string
		if email != "" {
			var user types.User
			err := tx.Unscoped().Where("email = ?", email).First(&user).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if hashErr != nil {
					return fmt.Errorf("hash OP administrator password: %w", hashErr)
				}
				user = types.User{
					ID:                  uuid.NewString(),
					Username:            email,
					Email:               email,
					PasswordHash:        string(hash),
					TenantID:            types.DefaultTenantID,
					IsActive:            true,
					CanAccessAllTenants: false,
					IsSystemAdmin:       true,
					AccessRole:          types.AccessRoleSuperAdmin,
					IsOpsAdmin:          true,
					PasswordChangedAt:   &now,
					CreatedAt:           now,
					UpdatedAt:           now,
				}
				if err := tx.Create(&user).Error; err != nil {
					return fmt.Errorf("create OP administrator: %w", err)
				}
			case err != nil:
				return fmt.Errorf("load OP administrator: %w", err)
			default:
				updates := map[string]interface{}{
					"tenant_id":              types.DefaultTenantID,
					"is_active":              true,
					"can_access_all_tenants": false,
					"is_system_admin":        true,
					"access_role":            types.AccessRoleSuperAdmin,
					"is_ops_admin":           true,
					"deleted_at":             nil,
					"updated_at":             now,
				}
				if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
					hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
					if hashErr != nil {
						return fmt.Errorf("hash OP administrator password: %w", hashErr)
					}
					updates["password_hash"] = string(hash)
					updates["password_changed_at"] = now
				}
				if err := tx.Unscoped().Model(&user).Updates(updates).Error; err != nil {
					return fmt.Errorf("update OP administrator: %w", err)
				}
			}
			ownerID = user.ID

			membership := types.TenantMember{
				UserID: user.ID, TenantID: types.DefaultTenantID, Role: types.TenantRoleOwner,
				Status: types.TenantMemberStatusActive, JoinedAt: now, CreatedAt: now, UpdatedAt: now,
			}
			var existing types.TenantMember
			if err := tx.Unscoped().Where("user_id = ? AND tenant_id = ?", user.ID, types.DefaultTenantID).First(&existing).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&membership).Error; err != nil {
					return fmt.Errorf("create OP administrator membership: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("load OP administrator membership: %w", err)
			} else if err := tx.Unscoped().Model(&existing).Updates(map[string]interface{}{
				"role": types.TenantRoleOwner, "status": types.TenantMemberStatusActive,
				"deleted_at": nil, "updated_at": now,
			}).Error; err != nil {
				return fmt.Errorf("update OP administrator membership: %w", err)
			}

			var profile types.SDPivotUserProfile
			if err := tx.Unscoped().Where("user_id = ?", user.ID).First(&profile).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				profile = types.SDPivotUserProfile{UserID: user.ID, Nickname: user.Username, Status: "active", CreatedAt: now, UpdatedAt: now}
				if err := tx.Create(&profile).Error; err != nil {
					return fmt.Errorf("create OP administrator profile: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("load OP administrator profile: %w", err)
			} else if err := tx.Unscoped().Model(&profile).Updates(map[string]interface{}{
				"status": "active", "deleted_at": nil, "updated_at": now,
			}).Error; err != nil {
				return fmt.Errorf("update OP administrator profile: %w", err)
			}
		} else {
			var owner types.User
			if err := tx.Where("tenant_id = ? AND is_active = ?", types.DefaultTenantID, true).
				Order("created_at ASC, id ASC").First(&owner).Error; err == nil {
				ownerID = owner.ID
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("load canonical organization owner: %w", err)
			}
		}

		org := types.Organization{
			ID: types.DefaultOrganizationID, Name: "默认组织", Description: "SDPivot OP default organization",
			OwnerID: ownerID, OwnerTenantID: types.DefaultTenantID, InviteCode: "SDP-DEFAULT",
			MemberLimit: 200, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Unscoped().Where("id = ?", types.DefaultOrganizationID).FirstOrCreate(&org).Error; err != nil {
			return fmt.Errorf("ensure canonical organization: %w", err)
		}
		orgUpdates := map[string]interface{}{
			"owner_tenant_id": types.DefaultTenantID, "deleted_at": nil, "updated_at": now,
		}
		if ownerID != "" {
			orgUpdates["owner_id"] = ownerID
		}
		if err := tx.Unscoped().Model(&org).Updates(orgUpdates).Error; err != nil {
			return fmt.Errorf("repair canonical organization: %w", err)
		}

		orgTenantMember := types.OrganizationTenantMember{
			ID: uuid.NewString(), OrganizationID: types.DefaultOrganizationID, TenantID: types.DefaultTenantID,
			Role: types.OrgRoleAdmin, RepresentativeUserID: ownerID, JoinedAt: &now, CreatedAt: now, UpdatedAt: now,
		}
		var existingOrgTenantMember types.OrganizationTenantMember
		if err := tx.Where("organization_id = ? AND tenant_id = ?", types.DefaultOrganizationID, types.DefaultTenantID).
			First(&existingOrgTenantMember).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&orgTenantMember).Error; err != nil {
				return fmt.Errorf("create canonical organization tenant membership: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("load canonical organization tenant membership: %w", err)
		} else {
			updates := map[string]interface{}{"role": types.OrgRoleAdmin, "updated_at": now}
			if ownerID != "" {
				updates["representative_user_id"] = ownerID
			}
			if err := tx.Model(&existingOrgTenantMember).Updates(updates).Error; err != nil {
				return fmt.Errorf("repair canonical organization tenant membership: %w", err)
			}
		}

		orgExt := types.OrgExt{
			OrgID: types.DefaultOrganizationID, TenantID: types.DefaultTenantID, AuthStatus: "active",
			AuthType: "op", CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Where("org_id = ?", types.DefaultOrganizationID).FirstOrCreate(&orgExt).Error; err != nil {
			return fmt.Errorf("ensure canonical organization extension: %w", err)
		}
		if err := tx.Model(&orgExt).Updates(map[string]interface{}{
			"tenant_id": types.DefaultTenantID, "auth_status": "active", "auth_type": "op", "updated_at": now,
		}).Error; err != nil {
			return fmt.Errorf("repair canonical organization extension: %w", err)
		}

		var users []types.User
		if err := tx.Where("tenant_id = ? AND is_active = ?", types.DefaultTenantID, true).Find(&users).Error; err != nil {
			return fmt.Errorf("list canonical organization accounts: %w", err)
		}
		for _, account := range users {
			role := "member"
			if account.ID == ownerID {
				role = "owner"
			}
			var orgMember types.SDPivotOrgMember
			if err := tx.Where("org_id = ? AND user_id = ?", types.DefaultOrganizationID, account.ID).
				First(&orgMember).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				orgMember = types.SDPivotOrgMember{
					ID: uuid.NewString(), OrgID: types.DefaultOrganizationID, UserID: account.ID,
					Role: role, Status: "active", JoinedAt: now, CreatedAt: now, UpdatedAt: now,
				}
				if err := tx.Create(&orgMember).Error; err != nil {
					return fmt.Errorf("create canonical organization account membership: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("load canonical organization account membership: %w", err)
			} else if err := tx.Model(&orgMember).Updates(map[string]interface{}{
				"role": role, "status": "active", "updated_at": now,
			}).Error; err != nil {
				return fmt.Errorf("repair canonical organization account membership: %w", err)
			}
		}

		if err := tx.Model(&types.KnowledgeSpace{}).
			Where("tenant_id = ? AND org_id IS NULL", types.DefaultTenantID).
			Update("org_id", types.DefaultOrganizationID).Error; err != nil {
			return fmt.Errorf("link canonical organization spaces: %w", err)
		}
		return nil
	})
}

type dbPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func validateStandaloneConfig(product *config.ProductConfig) error {
	if product == nil || !product.OPMode {
		return nil
	}
	if strings.TrimSpace(getEnvFallback("SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET", "")) == "" {
		return fmt.Errorf("OP mode requires SDP_JWT_SECRET or SMARTKNORA_JWT_SECRET")
	}
	if strings.TrimSpace(getEnvFallback("SDP_DB_PASSWORD", "SMART_DB_PASSWORD", "")) == "" {
		return fmt.Errorf("OP mode requires SDP_DB_PASSWORD or SMART_DB_PASSWORD")
	}
	return nil
}

func loadDBPoolConfig() (dbPoolConfig, error) {
	maxOpen, err := parseEnvInt("SDP_DB_MAX_OPEN_CONNS", 10, false)
	if err != nil {
		return dbPoolConfig{}, err
	}
	maxIdle, err := parseEnvInt("SDP_DB_MAX_IDLE_CONNS", 5, true)
	if err != nil {
		return dbPoolConfig{}, err
	}
	if maxIdle > maxOpen {
		return dbPoolConfig{}, fmt.Errorf("SDP_DB_MAX_IDLE_CONNS must not exceed SDP_DB_MAX_OPEN_CONNS")
	}
	lifetimeValue := strings.TrimSpace(getEnv("SDP_DB_CONN_MAX_LIFETIME", "10m"))
	lifetime, err := time.ParseDuration(lifetimeValue)
	if err != nil || lifetime <= 0 {
		return dbPoolConfig{}, fmt.Errorf("SDP_DB_CONN_MAX_LIFETIME must be a positive duration")
	}
	return dbPoolConfig{MaxOpenConns: maxOpen, MaxIdleConns: maxIdle, ConnMaxLifetime: lifetime}, nil
}

func parseEnvInt(name string, defaultValue int, allowZero bool) (int, error) {
	value := strings.TrimSpace(getEnv(name, strconv.Itoa(defaultValue)))
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 || (!allowZero && parsed == 0) {
		if allowZero {
			return 0, fmt.Errorf("%s must be an integer greater than or equal to zero", name)
		}
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}

func loadReadyTimeout() (time.Duration, error) {
	value := strings.TrimSpace(getEnv("SDP_READY_TIMEOUT", "3s"))
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return 0, fmt.Errorf("SDP_READY_TIMEOUT must be a positive duration")
	}
	return timeout, nil
}

func newReadinessCheck(
	pingDB func(context.Context) error,
	pingRedis func(context.Context) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		if pingDB != nil {
			if err := pingDB(ctx); err != nil {
				return err
			}
		}
		if pingRedis != nil {
			return pingRedis(ctx)
		}
		return nil
	}
}

func closeDependencies(db interface{ Close() error }, closeRedis func() error) {
	if closeRedis != nil {
		if err := closeRedis(); err != nil {
			log.Printf("[Redis] Close warning: %v", err)
		}
	}
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("[DB] Close warning: %v", err)
		}
	}
}

func registerTopLevelHealth(
	r *gin.Engine,
	readyTimeout time.Duration,
	readinessCheck func(context.Context) error,
) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "sdp", "version": "2.0.0"})
	})
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), readyTimeout)
		defer cancel()
		if readinessCheck != nil && readinessCheck(ctx) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}

func getAllowedOrigins() []string {
	value := getEnvFallback("SDP_ALLOWED_ORIGINS", "SMARTKNORA_ALLOWED_ORIGINS", "")
	if value == "" {
		return []string{
			// Dev server (43.133.61.77)
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://43.133.61.77:3099",
			// Test server (47.110.51.90)
			"http://47.110.51.90:3099",
			"http://localhost:3099",
			"http://127.0.0.1:3099",
		}
	}

	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvFallback(primary, fallback, def string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return getEnv(fallback, def)
}
