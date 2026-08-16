package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appLogger "github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
)

const sdpivotAPITokenAuthenticatedKey = "sdpivot_api_token_authenticated"

// RequireAPIToken authenticates sdp_ Bearer credentials. Non-API Bearer tokens
// continue to the JWT middleware so the protected route group supports both.
func RequireAPIToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer sdp_") {
			c.Next()
			return
		}
		raw := strings.TrimPrefix(authHeader, "Bearer ")
		hash := sha256.Sum256([]byte(raw))
		tokenHash := hex.EncodeToString(hash[:])
		type tokenRecord struct {
			ID         string
			TenantID   uint64
			CreatedBy  string
			UserID     string
			ExpiresAt  *time.Time
			RevokedAt  *time.Time
			Scopes     json.RawMessage
			AllowedIPs    json.RawMessage `gorm:"column:allowed_ips"`
			ScopeEnforced bool            `gorm:"column:scope_enforced"`
		}
		var token tokenRecord
		if err := db.WithContext(c.Request.Context()).Table("api_tokens").
			Where("token_hash = ?", tokenHash).First(&token).Error; err != nil || token.RevokedAt != nil || (token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now())) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired api token"})
			c.Abort()
			return
		}
		allowedIPs, err := parseTokenStringList(token.AllowedIPs)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "api token ip whitelist is invalid"})
			c.Abort()
			return
		}
		if len(allowedIPs) > 0 && !utils.IPAllowed(c.ClientIP(), allowedIPs) {
			c.JSON(http.StatusForbidden, gin.H{"error": "api token is not allowed from this ip"})
			c.Abort()
			return
		}
		scopes, err := parseTokenScopes(token.Scopes)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "api token scopes are invalid"})
			c.Abort()
			return
		}
		if !token.ScopeEnforced {
			appLogger.Warnf(c.Request.Context(), "legacy api token %s has no scopes; preserving owner-role permissions", token.ID)
		} else {
			c.Set(sdpivotAPITokenScopesKey, scopes)
		}
		userID := token.CreatedBy
		if userID == "" {
			userID = token.UserID
		}
		var user types.User
		if err := db.WithContext(c.Request.Context()).Select("id, tenant_id, access_role, department_id, is_active").Where("id = ? AND tenant_id = ?", userID, token.TenantID).First(&user).Error; err != nil || !user.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "api token owner is unavailable"})
			c.Abort()
			return
		}
		now := time.Now()
		_ = db.WithContext(c.Request.Context()).Table("api_tokens").Where("id = ?", token.ID).Update("last_used_at", now).Error
		c.Set("user_id", user.ID)
		c.Set("tenant_id", token.TenantID)
		c.Set("role", string(types.NormalizeAccessRole(string(user.AccessRole))))
		if user.DepartmentID != nil {
			c.Set("department_id", *user.DepartmentID)
		}
		c.Set(sdpivotAPITokenAuthenticatedKey, true)
		c.Next()
	}
}

func parseTokenStringList(raw json.RawMessage) ([]string, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" || value == "[]" {
		return nil, nil
	}
	var entries []string
	if strings.HasPrefix(value, "[") {
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, err
		}
	} else {
		value = strings.Trim(value, `"`)
		entries = strings.Split(value, ",")
	}
	for index := range entries {
		entries[index] = strings.TrimSpace(entries[index])
	}
	return entries, nil
}

func parseTokenScopes(raw json.RawMessage) (map[Permission]struct{}, error) {
	entries, err := parseTokenStringList(raw)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	scopes := make(map[Permission]struct{}, len(entries))
	for _, entry := range entries {
		permission := Permission(entry)
		switch permission {
		case PermissionUserRoleAssign, PermissionDepartmentManage, PermissionKnowledgeWrite, PermissionKnowledgeRead:
			scopes[permission] = struct{}{}
		default:
			return nil, fmt.Errorf("unknown api token scope %q", entry)
		}
	}
	return scopes, nil
}

func IsAPITokenAuthenticated(c *gin.Context) bool {
	value, _ := c.Get(sdpivotAPITokenAuthenticatedKey)
	authenticated, _ := value.(bool)
	return authenticated
}
