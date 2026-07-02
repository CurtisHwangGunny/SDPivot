package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SmartKnoraTenantContext creates a middleware that sets the PostgreSQL
// tenant context for RLS policies. Must run AFTER SmartKnoraAuth.
func SmartKnoraTenantContext(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := GetTenantID(c)
		if tenantID == 0 {
			c.Next()
			return
		}

		// Set PostgreSQL session variable for RLS policy
		// Convert tenantID to string to avoid encode type mismatch
		db.Exec("SELECT set_config('app.current_tenant_id', ?, false)", fmt.Sprintf("%d", tenantID))

		c.Next()
	}
}
