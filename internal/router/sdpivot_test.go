package router

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/gin-gonic/gin"
)

func newSDPivotTestEngine(t *testing.T, product *config.ProductConfig) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewSDPivotRouter(SDPivotRouterParams{Config: &config.Config{Product: product}}).RegisterRoutes(r)
	return r
}

func routeSet(r *gin.Engine, prefix string) []string {
	var routes []string
	for _, route := range r.Routes() {
		if strings.HasPrefix(route.Path, prefix) {
			routes = append(routes, route.Method+" "+strings.TrimPrefix(route.Path, prefix))
		}
	}
	sort.Strings(routes)
	return routes
}

func hasRoute(r *gin.Engine, method, path string) bool {
	for _, route := range r.Routes() {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}

func TestSDPivotRoutesCanonicalAlwaysRegistered(t *testing.T) {
	for _, cfg := range []*config.ProductConfig{
		nil,
		{Brand: "sdpivot", EnableLegacyAlias: false},
		{Brand: "sdpivot", EnableLegacyAlias: true},
	} {
		r := newSDPivotTestEngine(t, cfg)
		if !hasRoute(r, http.MethodGet, "/api/v1/sdp/health") {
			t.Fatal("canonical health route missing")
		}
	}
}

func TestSDPivotLegacyAliasToggleAndParity(t *testing.T) {
	disabled := newSDPivotTestEngine(t, &config.ProductConfig{Brand: "sdpivot"})
	if len(routeSet(disabled, "/api/v1/smartknora")) != 0 {
		t.Fatal("legacy routes registered when disabled")
	}

	enabled := newSDPivotTestEngine(t, config.DefaultProductConfig())
	canonical := routeSet(enabled, "/api/v1/sdp")
	legacy := routeSet(enabled, "/api/v1/smartknora")
	if len(canonical) == 0 || len(canonical) != len(legacy) {
		t.Fatalf("route counts differ: canonical=%d legacy=%d", len(canonical), len(legacy))
	}
	for i := range canonical {
		if canonical[i] != legacy[i] {
			t.Fatalf("route mismatch at %d: %q != %q", i, canonical[i], legacy[i])
		}
	}
}

func TestSDPivotRepresentativeRoutes(t *testing.T) {
	r := newSDPivotTestEngine(t, config.DefaultProductConfig())
	for _, prefix := range []string{"/api/v1/sdp", "/api/v1/smartknora"} {
		for _, route := range []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/health"},
			{http.MethodPost, "/ops/login"},
			{http.MethodGet, "/ops/dashboard"},
			{http.MethodGet, "/ops/announcements/active"},
		} {
			if !hasRoute(r, route.method, prefix+route.path) {
				t.Fatalf("missing route %s %s%s", route.method, prefix, route.path)
			}
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/sdp/health", nil)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("canonical health status = %d", response.Code)
	}
}
