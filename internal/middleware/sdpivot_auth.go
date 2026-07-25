package middleware

import (
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

// Permission names a protected product action.
type Permission string

const (
	PermissionUserRoleAssign Permission = "user.role.assign"
	PermissionDepartmentManage Permission = "department.manage"
	PermissionKnowledgeWrite Permission = "knowledge.write"
	PermissionKnowledgeRead Permission = "knowledge.read"
)

var accessRolePermissions = map[types.AccessRole]map[Permission]struct{}{
	types.AccessRoleSuperAdmin: {
		PermissionUserRoleAssign: {}, PermissionDepartmentManage: {},
		PermissionKnowledgeWrite: {}, PermissionKnowledgeRead: {},
	},
	types.AccessRoleDepartmentAdmin: {
		PermissionDepartmentManage: {}, PermissionKnowledgeWrite: {}, PermissionKnowledgeRead: {},
	},
	types.AccessRoleKnowledgeEditor: {
		PermissionKnowledgeWrite: {}, PermissionKnowledgeRead: {},
	},
	types.AccessRoleKnowledgeViewer: {
		PermissionKnowledgeRead: {},
	},
}

// HasPermission checks a product role against the permission matrix.
func HasPermission(role string, permission Permission) bool {
	_, ok := accessRolePermissions[types.NormalizeAccessRole(role)][permission]
	return ok
}

// RequirePermission rejects callers whose JWT role lacks permission.
func RequirePermission(permission Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasPermission(GetRole(c), permission) {
			c.Next()
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission", "permission": permission})
		c.Abort()
	}
}

// SDPivotAuth creates a middleware that validates JWT access tokens
// and sets user context for SDPivot routes.
func SDPivotAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Check for public routes
		if isSDPivotPublicPath(c.Request.URL.Path, c.Request.Method) {
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
		if claims.Role == "ops_admin" && !isSDPivotOpsPath(c.Request.URL.Path) {
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

// SDPivotPublicPaths defines routes that skip authentication.
var sdPivotPublicPaths = map[string][]string{
	"/api/v1/sdp/auth/register":        {"POST"},
	"/api/v1/sdp/auth/login":           {"POST"},
	"/api/v1/sdp/auth/refresh":         {"POST"},
	"/api/v1/sdp/health":               {"GET"},
	"/api/v1/smartknora/auth/register": {"POST"},
	"/api/v1/smartknora/auth/login":    {"POST"},
	"/api/v1/smartknora/auth/refresh":  {"POST"},
	"/api/v1/smartknora/health":        {"GET"},
}

func isSDPivotOpsPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/sdp/ops") || strings.HasPrefix(path, "/api/v1/smartknora/ops")
}

func isSDPivotPublicPath(path string, method string) bool {
	methods, ok := sdPivotPublicPaths[path]
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

// GetUserID extracts user_id from Gin context (set by SDPivotAuth middleware).
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
