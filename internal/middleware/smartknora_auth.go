package middleware

import (
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/gin-gonic/gin"
)

// SmartKnoraAuth creates a middleware that validates JWT access tokens
// and sets user context for smartKnora routes.
func SmartKnoraAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Check for public routes
		if isSmartKnoraPublicPath(c.Request.URL.Path, c.Request.Method) {
			c.Next()
			return
		}

		// Extract Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate JWT
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Ops-admin tokens are scoped to /ops APIs only. Normal SaaS APIs must not
		// accept them, otherwise the operations plane and tenant plane are mixed.
		if claims.Role == "ops_admin" && !strings.HasPrefix(c.Request.URL.Path, "/api/v1/smartknora/ops") {
			c.JSON(http.StatusForbidden, gin.H{"error": "ops admin token is not allowed on tenant APIs"})
			c.Abort()
			return
		}

		// Set context values for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("role", claims.Role)
		c.Set("jti", claims.JTI)

		c.Next()
	}
}

// SmartKnoraPublicPaths defines routes that skip authentication.
var smartKnoraPublicPaths = map[string][]string{
	"/api/v1/smartknora/auth/register": {"POST"},
	"/api/v1/smartknora/auth/login":    {"POST"},
	"/api/v1/smartknora/auth/refresh":  {"POST"},
	"/api/v1/smartknora/health":        {"GET"},
}

func isSmartKnoraPublicPath(path string, method string) bool {
	methods, ok := smartKnoraPublicPaths[path]
	if !ok {
		return false
	}
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

// GetUserID extracts user_id from Gin context (set by SmartKnoraAuth middleware).
func GetUserID(c *gin.Context) string {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// GetTenantID extracts tenant_id from Gin context.
func GetTenantID(c *gin.Context) uint64 {
	if v, ok := c.Get("tenant_id"); ok {
		if id, ok := v.(uint64); ok {
			return id
		}
	}
	return 0
}

// GetRole extracts role from Gin context.
func GetRole(c *gin.Context) string {
	if v, ok := c.Get("role"); ok {
		if role, ok := v.(string); ok {
			return role
		}
	}
	return ""
}
