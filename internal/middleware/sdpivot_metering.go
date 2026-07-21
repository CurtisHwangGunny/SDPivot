package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TokenMeteringMiddleware intercepts API responses to record token usage.
// This is a placeholder that logs API path and basic request info.
// Actual token counting happens in the LLM pipeline.
func TokenMeteringMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Only meter AI-related endpoints
		path := c.Request.URL.Path
		if !isMeteredPath(path) {
			return
		}

		userID := GetUserID(c)
		tenantID := GetTenantID(c)

		// Try to extract token usage from response context
		// (set by LLM pipeline handlers)
		promptTokens, _ := c.Get("prompt_tokens")
		completionTokens, _ := c.Get("completion_tokens")

		pTokens, _ := promptTokens.(int)
		cTokens, _ := completionTokens.(int)

		if pTokens == 0 && cTokens == 0 {
			return // No token data available
		}

		usage := types.SDPivotTokenUsage{
			UserID:           &userID,
			TenantID:         tenantID,
			PromptTokens:     pTokens,
			CompletionTokens: cTokens,
			TotalTokens:      pTokens + cTokens,
			APIPath:          path,
			CreatedAt:        start,
		}

		// Record async to avoid blocking response
		go func() {
			db.Create(&usage)
		}()
	}
}

// isMeteredPath checks if the API path should be metered.
func isMeteredPath(path string) bool {
	meteredPaths := []string{
		"/api/v1/sdp/qa/",             // AI Q&A sessions
		"/api/v1/sdp/writing/",        // AI writing assistant
		"/api/v1/smartknora/qa/",      // Legacy AI Q&A alias
		"/api/v1/smartknora/writing/", // Legacy AI writing alias
		"/api/v1/completion",          // LLM completion
	}
	for _, p := range meteredPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// CaptureRequestBody reads and restores the request body for logging.
func CaptureRequestBody(c *gin.Context) []byte {
	if c.Request.Body == nil {
		return nil
	}
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body
}
