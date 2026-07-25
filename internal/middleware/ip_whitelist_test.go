package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type ipWhitelistSettings struct {
	interfaces.SystemSettingService
	entries []string
}

func (s *ipWhitelistSettings) GetStringList(context.Context, string, string, []string) []string {
	return s.entries
}

func TestIPWhitelistEnforcement(t *testing.T) {
	tests := []struct {
		name       string
		entries    []string
		remoteAddr string
		want       int
	}{
		{"disabled when empty", nil, "203.0.113.10:1234", http.StatusOK},
		{"exact IP allowed", []string{"203.0.113.10"}, "203.0.113.10:1234", http.StatusOK},
		{"CIDR allowed", []string{"203.0.113.0/24"}, "203.0.113.10:1234", http.StatusOK},
		{"outside whitelist rejected", []string{"198.51.100.0/24"}, "203.0.113.10:1234", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			_ = r.SetTrustedProxies(nil)
			r.GET("/api", IPWhitelist(&ipWhitelistSettings{entries: tt.entries}), func(c *gin.Context) { c.Status(http.StatusOK) })
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api", nil)
			req.RemoteAddr = tt.remoteAddr
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Code)
		})
	}
}
