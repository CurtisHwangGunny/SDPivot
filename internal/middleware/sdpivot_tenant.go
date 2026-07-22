package middleware

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const sdPivotTenantDBKey = "sdpivot_tenant_db"

func sdPivotTenantFallbackContext(tenantID uint64) (string, []interface{}) {
	return "SELECT set_config('app.current_tenant_id', ?, true), set_config('app.is_ops_admin', 'false', true)", []interface{}{fmt.Sprintf("%d", tenantID)}
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

		isOpsAdmin := GetRole(c) == "ops_admin"
		tx := baseDB.WithContext(c.Request.Context()).Begin()
		if tx.Error != nil {
			c.JSON(500, gin.H{"error": "failed to start tenant database transaction"})
			c.Abort()
			return
		}

		if err := tx.Exec("SELECT set_tenant_context(?, ?)", tenantID, isOpsAdmin).Error; err != nil {
			fallbackSQL, fallbackArgs := sdPivotTenantFallbackContext(tenantID)
			if fallbackErr := tx.Exec(fallbackSQL, fallbackArgs...).Error; fallbackErr != nil {
				_ = tx.Rollback().Error
				log.Printf("failed to set tenant database context: %v", fallbackErr)
				c.JSON(500, gin.H{"error": "failed to set tenant database context"})
				c.Abort()
				return
			}
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
