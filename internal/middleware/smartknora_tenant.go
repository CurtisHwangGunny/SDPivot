package middleware

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SmartKnoraTenantContext creates a middleware that sets the PostgreSQL
// tenant context for RLS policies. Must run AFTER SmartKnoraAuth.
func SmartKnoraTenantContext(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := GetTenantID(c)
		if tenantID == 0 {
			// TenantID=0 is expected for public routes (register/login).
			// These endpoints don't need RLS tenant isolation.
			c.Next()
			return
		}

		// Set PostgreSQL session variable for RLS policy
		// This ensures all subsequent queries in this request
		// are filtered by tenant_id
		// TODO: Connection pool race condition — session-level set_config can leak
		// across requests on the same connection. Fix: use per-request transactions
		// (db.Session(&gorm.Session{NewDB: true}).Transaction(...)) or dedicated connections.
		db.Exec("SELECT set_config('app.current_tenant_id', ?, false)", tenantID)

		c.Next()
	}
}
