package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterSDPivotHealthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerSDPivotHealth(router.Group("/api/v1/sdp"))
	registerSDPivotHealth(router.Group("/api/v1/smartknora"))

	for _, path := range []string{"/api/v1/sdp/health", "/api/v1/smartknora/health"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["service"] != "sdp" {
				t.Fatalf("expected service sdp, got %v", body["service"])
			}
		})
	}
}
