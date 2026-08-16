package middleware

import (
	"fmt"
	"log"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const sdPivotTenantDBKey = "sdpivot_tenant_db"

// SDPivotTenantContext binds a request transaction and normalized product role
// to PostgreSQL GUCs. Existing official tenant RLS remains active; the role GUC
// is additive and lets OP policies avoid historical role-string checks.
func SDPivotTenantContext(baseDB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetUint64(types.TenantIDContextKey.String())
		if tenantID == 0 {
			c.Next()
			return
		}
		tx := baseDB.WithContext(c.Request.Context()).Begin()
		if tx.Error != nil {
			c.JSON(500, gin.H{"error": "failed to start tenant database transaction"})
			c.Abort()
			return
		}
		role := types.NormalizeAccessRole(string(types.AccessRoleFromContext(c.Request.Context())))
		if err := tx.Exec("SELECT set_config('app.current_tenant_id', ?, true), set_config('app.access_role', ?, true)", fmt.Sprintf("%d", tenantID), string(role)).Error; err != nil {
			_ = tx.Rollback().Error
			log.Printf("failed to set tenant database context: %v", err)
			c.JSON(500, gin.H{"error": "failed to set tenant database context"})
			c.Abort()
			return
		}
		c.Set(sdPivotTenantDBKey, tx)
		c.Next()
		if len(c.Errors) > 0 || c.Writer.Status() >= 400 {
			_ = tx.Rollback().Error
			return
		}
		_ = tx.Commit().Error
	}
}

func TenantDB(c *gin.Context, fallback *gorm.DB) *gorm.DB {
	if c != nil {
		if tx, ok := c.Get(sdPivotTenantDBKey); ok {
			if db, ok := tx.(*gorm.DB); ok && db != nil {
				return db
			}
		}
		return fallback.WithContext(c.Request.Context())
	}
	return fallback
}
