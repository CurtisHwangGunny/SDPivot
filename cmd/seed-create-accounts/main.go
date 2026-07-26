package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID                  string `gorm:"type:varchar(36);primaryKey"`
	Username            string
	Email               string
	PasswordHash        string
	TenantID            uint64
	IsActive            bool
	CanAccessAllTenants bool
	IsSystemAdmin       bool
	IsOpsAdmin          bool
	AccessRole          string
	MustChangePassword  bool
	PasswordChangedAt   *time.Time
	PasswordExpiresAt   *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (User) TableName() string { return "users" }

type Profile struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	UserID    string
	Nickname  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Profile) TableName() string { return "smartknora_user_profiles" }

type Account struct {
	Email              string
	Username           string
	Nickname           string
	Password           string
	TenantID           uint64
	IsOpsAdmin         bool
	IsSystemAdmin      bool
	MustChangePassword bool
	IsActive           bool
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseBool(value string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		log.Fatalf("invalid bool value %q", value)
	}
	return def
}

func required(row map[string]string, key string) string {
	v := strings.TrimSpace(row[key])
	if v == "" {
		log.Fatalf("missing required column %s", key)
	}
	return v
}

func loadAccounts(path string) ([]Account, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, errors.New("csv must contain header and at least one account row")
	}

	headers := records[0]
	var accounts []Account
	for i, rec := range records[1:] {
		row := map[string]string{}
		for idx, h := range headers {
			if idx < len(rec) {
				row[strings.TrimSpace(h)] = rec[idx]
			}
		}
		tenantID := types.DefaultTenantID
		if raw := strings.TrimSpace(row["tenant_id"]); raw != "" {
			parsed, err := strconv.ParseUint(raw, 10, 64)
			if err != nil || parsed == 0 {
				return nil, fmt.Errorf("row %d invalid tenant_id: %s", i+2, raw)
			}
			if parsed != types.DefaultTenantID {
				return nil, fmt.Errorf("row %d tenant_id must be %d in OP mode", i+2, types.DefaultTenantID)
			}
			tenantID = parsed
		}
		acc := Account{
			Email:              required(row, "email"),
			Username:           required(row, "username"),
			Nickname:           strings.TrimSpace(row["nickname"]),
			Password:           required(row, "password"),
			TenantID:           tenantID,
			IsOpsAdmin:         parseBool(row["is_ops_admin"], true),
			IsSystemAdmin:      parseBool(row["is_system_admin"], false),
			MustChangePassword: parseBool(row["must_change_password"], true),
			IsActive:           parseBool(row["is_active"], true),
		}
		if len(acc.Password) < 8 {
			return nil, fmt.Errorf("row %d password must be at least 8 chars", i+2)
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func main() {
	csvPath := flag.String("file", "", "CSV account file")
	dryRun := flag.Bool("dry-run", false, "Validate only; do not write database")
	flag.Parse()
	if *csvPath == "" {
		log.Fatal("missing --file")
	}

	accounts, err := loadAccounts(*csvPath)
	if err != nil {
		log.Fatal(err)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		env("SMART_DB_HOST", "localhost"), env("SMART_DB_PORT", "5432"), env("SMART_DB_USER", "postgres"), env("SMART_DB_PASSWORD", ""), env("SMART_DB_NAME", "WeKnora"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	fmt.Printf("Loaded %d account(s). dry_run=%v\n", len(accounts), *dryRun)
	for _, acc := range accounts {
		now := time.Now()
		expires := now.Add(90 * 24 * time.Hour)

		var existing User
		queryErr := db.Where("email = ?", acc.Email).First(&existing).Error
		if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
			log.Fatalf("query user %s: %v", acc.Email, queryErr)
		}

		if *dryRun {
			if errors.Is(queryErr, gorm.ErrRecordNotFound) {
				fmt.Printf("DRY-RUN create ops account: email=%s username=%s tenant_id=%d ops=%v system=%v\n", acc.Email, acc.Username, acc.TenantID, acc.IsOpsAdmin, acc.IsSystemAdmin)
			} else {
				fmt.Printf("DRY-RUN update ops account: email=%s username=%s tenant_id=%d ops=%v system=%v\n", acc.Email, acc.Username, acc.TenantID, acc.IsOpsAdmin, acc.IsSystemAdmin)
			}
			continue
		}

		hash, passwordChanged, err := passwordHash(existing.PasswordHash, acc.Password)
		if err != nil {
			log.Fatalf("hash password for %s: %v", acc.Email, err)
		}

		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			user := User{
				ID:                  uuid.New().String(),
				Username:            acc.Username,
				Email:               acc.Email,
				PasswordHash:        string(hash),
				TenantID:            acc.TenantID,
				IsActive:            acc.IsActive,
				CanAccessAllTenants: false,
				IsSystemAdmin:       acc.IsSystemAdmin,
				IsOpsAdmin:          acc.IsOpsAdmin,
				AccessRole:          accountAccessRole(acc),
				MustChangePassword:  acc.MustChangePassword,
				PasswordChangedAt:   &now,
				PasswordExpiresAt:   &expires,
				CreatedAt:           now,
				UpdatedAt:           now,
			}
			if err := db.Create(&user).Error; err != nil {
				log.Fatalf("create user %s: %v", acc.Email, err)
			}
			profile := Profile{ID: uuid.New().String(), UserID: user.ID, Nickname: acc.Nickname, Status: "active", CreatedAt: now, UpdatedAt: now}
			if profile.Nickname == "" {
				profile.Nickname = acc.Username
			}
			if err := db.Create(&profile).Error; err != nil {
				log.Printf("warning: create profile for %s: %v", acc.Email, err)
			}
			fmt.Printf("CREATED ops account: email=%s username=%s tenant_id=%d ops=%v system=%v\n", acc.Email, acc.Username, acc.TenantID, acc.IsOpsAdmin, acc.IsSystemAdmin)
		} else {
			updates := map[string]interface{}{
				"username":               acc.Username,
				"tenant_id":              acc.TenantID,
				"is_active":              acc.IsActive,
				"can_access_all_tenants": false,
				"is_system_admin":        acc.IsSystemAdmin,
				"is_ops_admin":           acc.IsOpsAdmin,
				"access_role":            accountAccessRole(acc),
				"must_change_password":   acc.MustChangePassword,
				"updated_at":             now,
			}
			if passwordChanged {
				updates["password_hash"] = string(hash)
				updates["password_changed_at"] = now
				updates["password_expires_at"] = expires
			}
			if err := db.Model(&existing).Updates(updates).Error; err != nil {
				log.Fatalf("update user %s: %v", acc.Email, err)
			}
			if acc.Nickname != "" {
				db.Table("smartknora_user_profiles").Where("user_id = ?", existing.ID).Updates(map[string]interface{}{"nickname": acc.Nickname, "updated_at": now})
			}
			fmt.Printf("UPDATED ops account: email=%s username=%s tenant_id=%d ops=%v system=%v\n", acc.Email, acc.Username, acc.TenantID, acc.IsOpsAdmin, acc.IsSystemAdmin)
		}
	}
}

func accountAccessRole(acc Account) string {
	if acc.IsOpsAdmin || acc.IsSystemAdmin {
		return string(types.AccessRoleSuperAdmin)
	}
	return string(types.AccessRoleKnowledgeViewer)
}

func passwordHash(existingHash, password string) ([]byte, bool, error) {
	if existingHash != "" && bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(password)) == nil {
		return []byte(existingHash), false, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return hash, true, err
}
