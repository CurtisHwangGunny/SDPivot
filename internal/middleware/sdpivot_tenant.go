package middleware

import (
	"fmt"
	"log"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	sdPivotTenantDBKey     = "sdpivot_tenant_db"
	sdPivotTenantSavepoint = "sdpivot_tenant_context"
)

func sdPivotTenantFallbackContext(tenantID uint64) (string, []interface{}) {
	return "SELECT set_config('app.current_tenant_id', ?, true), set_config('app.is_ops_admin', 'false', true)", []interface{}{fmt.Sprintf("%d", tenantID)}
}

func configureSDPivotTenantContext(tx *gorm.DB, tenantID uint64, isOpsAdmin bool) error {
	if err := tx.Exec("SAVEPOINT " + sdPivotTenantSavepoint).Error; err != nil {
		return fmt.Errorf("create tenant context savepoint: %w", err)
	}

	if err := tx.Exec("SELECT set_tenant_context(?, ?)", tenantID, isOpsAdmin).Error; err == nil {
		if releaseErr := tx.Exec("RELEASE SAVEPOINT " + sdPivotTenantSavepoint).Error; releaseErr != nil {
			return fmt.Errorf("release tenant context savepoint: %w", releaseErr)
		}
		return nil
	}

	if err := tx.Exec("ROLLBACK TO SAVEPOINT " + sdPivotTenantSavepoint).Error; err != nil {
		return fmt.Errorf("restore tenant context savepoint: %w", err)
	}

	fallbackSQL, fallbackArgs := sdPivotTenantFallbackContext(tenantID)
	if err := tx.Exec(fallbackSQL, fallbackArgs...).Error; err != nil {
		return fmt.Errorf("set fallback tenant context: %w", err)
	}
	if err := tx.Exec("RELEASE SAVEPOINT " + sdPivotTenantSavepoint).Error; err != nil {
		return fmt.Errorf("release fallback tenant context savepoint: %w", err)
	}
	return nil
}

// SDPivotTenantContext creates a middleware that binds a request-scoped
// database transaction to the current request and sets PostgreSQL tenant
// context for RLS policies. Must run AFTER SDPivotAuth.
func SDPivotTenantContext(baseDB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := GetTenantID(c)
		if tenantID == 0 {
			c.Next()
			return
		}

		isOpsAdmin := GetAccessRole(c) == types.AccessRoleSuperAdmin
		tx := baseDB.WithContext(c.Request.Context()).Begin()
		if tx.Error != nil {
			c.JSON(500, gin.H{"error": "failed to start tenant database transaction"})
			c.Abort()
			return
		}

		if err := configureSDPivotTenantContext(tx, tenantID, isOpsAdmin); err != nil {
			_ = tx.Rollback().Error
			log.Printf("failed to set tenant database context: %v", err)
			c.JSON(500, gin.H{"error": "failed to set tenant database context"})
			c.Abort()
			return
		}

		c.Set(sdPivotTenantDBKey, tx)
		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback().Error
				panic(r)
			}
		}()

		c.Next()

		if len(c.Errors) > 0 || c.Writer.Status() >= 400 {
			_ = tx.Rollback().Error
			return
		}
		_ = tx.Commit().Error
	}
}

// TenantDB returns the request-scoped tenant transaction when available.
func TenantDB(c *gin.Context, fallback *gorm.DB) *gorm.DB {
	if c != nil {
		if tx, ok := c.Get(sdPivotTenantDBKey); ok {
			if tenantDB, ok := tx.(*gorm.DB); ok && tenantDB != nil {
				return tenantDB
			}
		}
		return fallback.WithContext(c.Request.Context())
	}
	return fallback
}
