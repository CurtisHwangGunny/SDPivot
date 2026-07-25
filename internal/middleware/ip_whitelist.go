package middleware

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// IPWhitelist enforces the dynamic API client allowlist after authentication.
// An empty list disables the restriction.
func IPWhitelist(settings interfaces.SystemSettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settings == nil {
			c.Next()
			return
		}
		entries := settings.GetStringList(c.Request.Context(), "security.ip_whitelist", "", []string{})
		if len(entries) == 0 || utils.IPAllowed(c.ClientIP(), entries) {
			c.Next()
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: client IP is not allowed"})
		c.Abort()
	}
}
