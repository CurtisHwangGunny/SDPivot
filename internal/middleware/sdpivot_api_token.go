package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
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
			ID        string
			TenantID  uint64
			CreatedBy string
			UserID    string
			ExpiresAt *time.Time
			RevokedAt *time.Time
		}
		var token tokenRecord
		if err := db.WithContext(c.Request.Context()).Table("api_tokens").
			Where("token_hash = ?", tokenHash).First(&token).Error; err != nil || token.RevokedAt != nil || (token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now())) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired api token"})
			c.Abort()
			return
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

func IsAPITokenAuthenticated(c *gin.Context) bool {
	value, _ := c.Get(sdpivotAPITokenAuthenticatedKey)
	authenticated, _ := value.(bool)
	return authenticated
}
