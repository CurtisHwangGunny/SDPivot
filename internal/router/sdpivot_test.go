package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
		for _, path := range []string{"/auth/register", "/ops/login"} {
			if hasRoute(r, http.MethodPost, prefix+path) {
				t.Fatalf("public route must not be registered: POST %s%s", prefix, path)
			}
			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest(http.MethodPost, prefix+path, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("POST %s%s status = %d, want 404", prefix, path, response.Code)
			}
		}
		for _, route := range []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/health"},
			{http.MethodPost, "/auth/logout"},
			{http.MethodPost, "/ops/users"},
			{http.MethodGet, "/ops/dashboard"},
			{http.MethodGet, "/ops/announcements/active"},
			{http.MethodGet, "/ops/usage-stats"},
			{http.MethodGet, "/admin/tags"},
			{http.MethodPost, "/admin/tags"},
			{http.MethodPut, "/admin/tags/:id"},
			{http.MethodDelete, "/admin/tags/:id"},
			{http.MethodGet, "/auth/me"},
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
	var health struct {
		OPMode bool `json:"op_mode"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.OPMode {
		t.Fatal("default product health unexpectedly reported OP mode")
	}
}

func TestSDPivotDepartmentRoutesRegistered(t *testing.T) {
	r := newSDPivotTestEngine(t, &config.ProductConfig{Brand: "sdpivot", OPMode: true})
	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/sdp/admin/departments"},
		{http.MethodPost, "/api/v1/sdp/admin/departments"},
		{http.MethodPut, "/api/v1/sdp/admin/departments/:id"},
		{http.MethodDelete, "/api/v1/sdp/admin/departments/:id"},
	} {
		if !hasRoute(r, route.method, route.path) {
			t.Errorf("missing route %s %s", route.method, route.path)
		}
	}
}

func TestSDPivotHealthReportsOPMode(t *testing.T) {
	r := newSDPivotTestEngine(t, &config.ProductConfig{Brand: "sdpivot", OPMode: true})
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/sdp/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("health status = %d", response.Code)
	}
	var health struct {
		OPMode bool `json:"op_mode"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if !health.OPMode {
		t.Fatal("OP product health did not report OP mode")
	}
}

func TestSDPivotCompatRoutesRequireJWT(t *testing.T) {
	r := newSDPivotTestEngine(t, config.DefaultProductConfig())
	for _, path := range []string{
		"/api/v1/sdp/auth/me",
		"/api/v1/sdp/admin/tags",
		"/api/v1/sdp/ops/usage-stats",
	} {
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s status = %d, want 401", path, response.Code)
		}
	}
}

func TestSDPivotOPModeDisablesSaaSOpsRoutes(t *testing.T) {
	r := newSDPivotTestEngine(t, &config.ProductConfig{Brand: "sdpivot", OPMode: true})
	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/ops/refresh"},
		{http.MethodGet, "/ops/config/trial"},
		{http.MethodPut, "/ops/config/trial"},
		{http.MethodPost, "/ops/announcements"},
		{http.MethodGet, "/ops/announcements"},
		{http.MethodDelete, "/ops/announcements/example"},
		{http.MethodGet, "/ops/announcements/active"},
	} {
		path := "/api/v1/sdp" + route.path
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(route.method, path, nil))
		if response.Code != http.StatusNotFound {
			t.Errorf("%s %s status = %d, want 404", route.method, path, response.Code)
		}
		assertOPFeatureDisabledResponse(t, response)
	}
}

func TestSDPivotOPModeDisablesOrganizationsBeforeValidationAndWrites(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sdpivot-op-disabled?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&types.Organization{}); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewSDPivotRouter(SDPivotRouterParams{
		DB:     db,
		Config: &config.Config{Product: &config.ProductConfig{Brand: "sdpivot", OPMode: true}},
	}).RegisterRoutes(r)

	for _, test := range []struct {
		path string
		body string
	}{
		{path: "/api/v1/sdp/organizations", body: `{"name":"must-not-exist"}`},
		{path: "/api/v1/sdp/organizations/join", body: `{"invite_code":"should-not-be-validated"}`},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf("POST %s status = %d, want 404", test.path, response.Code)
		}
		assertOPFeatureDisabledResponse(t, response)
	}

	var count int64
	if err := db.Model(&types.Organization{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("organization count = %d, want 0", count)
	}
}

func TestOPDisabledFeatureMiddlewareRunsBeforeDownstreamHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(opDisabledFeatureMiddleware())
	downstreamCalls := 0
	r.Any("/*path", func(c *gin.Context) {
		downstreamCalls++
		c.Status(http.StatusTeapot)
	})

	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/organizations"},
		{method: http.MethodPost, path: "/api/v1/sdp/organizations/join"},
		{method: http.MethodPost, path: "/api/v1/sdp/ops/refresh"},
		{method: http.MethodGet, path: "/api/v1/sdp/ops/config/trial"},
		{method: http.MethodGet, path: "/api/v1/sdp/ops/announcements"},
	} {
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest(test.method, test.path, strings.NewReader(`{"invalid":`)))
		if response.Code != http.StatusNotFound {
			t.Errorf("%s %s status = %d, want 404", test.method, test.path, response.Code)
		}
		assertOPFeatureDisabledResponse(t, response)
	}
	if downstreamCalls != 0 {
		t.Fatalf("downstream calls = %d, want 0", downstreamCalls)
	}
}

func assertOPFeatureDisabledResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode disabled response: %v", err)
	}
	if payload.Code != "FEATURE_DISABLED" || payload.Message != "feature is disabled in OP edition" {
		t.Fatalf("disabled response = %#v", payload)
	}
}
