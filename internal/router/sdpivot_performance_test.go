package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/gin-gonic/gin"
)

func BenchmarkSDPivotAPILatency(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSDPivotRouter(SDPivotRouterParams{
		Config: &config.Config{Product: config.DefaultProductConfig()},
	}).RegisterRoutes(router)

	for _, path := range []string{"/api/v1/sdp/health", "/api/v1/smartknora/health"} {
		b.Run(path, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				request := httptest.NewRequest(http.MethodGet, path, nil)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if response.Code != http.StatusOK {
					b.Fatalf("status = %d", response.Code)
				}
			}
		})
	}
}
