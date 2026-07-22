package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/gin-gonic/gin"
)

func TestValidateStandaloneConfig(t *testing.T) {
	for _, name := range []string{"SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET", "SDP_DB_PASSWORD", "SMART_DB_PASSWORD"} {
		t.Setenv(name, "")
	}
	product := &config.ProductConfig{OPMode: true}

	t.Setenv("SDP_DB_PASSWORD", "db-password")
	if err := validateStandaloneConfig(product); err == nil || !strings.Contains(err.Error(), "JWT") {
		t.Fatalf("expected missing JWT error, got %v", err)
	}

	t.Setenv("SDP_DB_PASSWORD", "")
	t.Setenv("SDP_JWT_SECRET", "jwt-secret")
	if err := validateStandaloneConfig(product); err == nil || !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("expected missing DB password error, got %v", err)
	}

	t.Setenv("SDP_JWT_SECRET", "")
	t.Setenv("SMARTKNORA_JWT_SECRET", "legacy-jwt")
	t.Setenv("SMART_DB_PASSWORD", "legacy-db")
	if err := validateStandaloneConfig(product); err != nil {
		t.Fatalf("legacy compatibility variables should pass: %v", err)
	}
}

func TestValidateStandaloneConfigNonOPAllowsDefaults(t *testing.T) {
	for _, name := range []string{"SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET", "SDP_DB_PASSWORD", "SMART_DB_PASSWORD"} {
		t.Setenv(name, "")
	}
	if err := validateStandaloneConfig(&config.ProductConfig{}); err != nil {
		t.Fatalf("non-OP mode should preserve defaults: %v", err)
	}
}

func TestLoadDBPoolConfig(t *testing.T) {
	setPoolEnv := func(open, idle, lifetime string) {
		t.Setenv("SDP_DB_MAX_OPEN_CONNS", open)
		t.Setenv("SDP_DB_MAX_IDLE_CONNS", idle)
		t.Setenv("SDP_DB_CONN_MAX_LIFETIME", lifetime)
	}

	t.Run("defaults", func(t *testing.T) {
		setPoolEnv("", "", "")
		cfg, err := loadDBPoolConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxOpenConns != 10 || cfg.MaxIdleConns != 5 || cfg.ConnMaxLifetime != 10*time.Minute {
			t.Fatalf("unexpected defaults: %+v", cfg)
		}
	})

	t.Run("valid overrides", func(t *testing.T) {
		setPoolEnv("8", "2", "30m")
		cfg, err := loadDBPoolConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxOpenConns != 8 || cfg.MaxIdleConns != 2 || cfg.ConnMaxLifetime != 30*time.Minute {
			t.Fatalf("unexpected overrides: %+v", cfg)
		}
	})

	for _, tc := range []struct {
		name     string
		open     string
		idle     string
		lifetime string
	}{
		{name: "invalid integer", open: "invalid", idle: "2", lifetime: "10m"},
		{name: "idle exceeds open", open: "2", idle: "3", lifetime: "10m"},
		{name: "invalid duration", open: "10", idle: "5", lifetime: "forever"},
		{name: "non-positive duration", open: "10", idle: "5", lifetime: "0s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setPoolEnv(tc.open, tc.idle, tc.lifetime)
			if _, err := loadDBPoolConfig(); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestLoadReadyTimeout(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("SDP_READY_TIMEOUT", "")
		timeout, err := loadReadyTimeout()
		if err != nil || timeout != 3*time.Second {
			t.Fatalf("timeout=%v err=%v", timeout, err)
		}
	})
	t.Run("override", func(t *testing.T) {
		t.Setenv("SDP_READY_TIMEOUT", "750ms")
		timeout, err := loadReadyTimeout()
		if err != nil || timeout != 750*time.Millisecond {
			t.Fatalf("timeout=%v err=%v", timeout, err)
		}
	})
	for _, value := range []string{"invalid", "0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("SDP_READY_TIMEOUT", value)
			if _, err := loadReadyTimeout(); err == nil {
				t.Fatal("expected readiness timeout error")
			}
		})
	}
}

func TestTopLevelHealthAndReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	internalError := "password=super-secret internal connection failure"
	r := gin.New()
	registerTopLevelHealth(r, time.Second, func(context.Context) error {
		return errors.New(internalError)
	})

	health := httptest.NewRecorder()
	r.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}

	ready := httptest.NewRecorder()
	r.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", ready.Code)
	}
	if strings.Contains(ready.Body.String(), internalError) || strings.Contains(ready.Body.String(), "super-secret") {
		t.Fatalf("readiness response leaked internal error: %s", ready.Body.String())
	}
}

func TestTopLevelReadinessUsesTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerTopLevelHealth(r, 20*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", response.Code)
	}
}

func TestReadinessDependencyOrderAndShortCircuit(t *testing.T) {
	var calls []string
	check := newReadinessCheck(
		func(context.Context) error {
			calls = append(calls, "db")
			return nil
		},
		func(context.Context) error {
			calls = append(calls, "redis")
			return nil
		},
	)
	if err := check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "db,redis" {
		t.Fatalf("unexpected readiness order: %v", calls)
	}

	calls = nil
	check = newReadinessCheck(
		func(context.Context) error {
			calls = append(calls, "db")
			return errors.New("database unavailable")
		},
		func(context.Context) error {
			calls = append(calls, "redis")
			return nil
		},
	)
	if err := check(context.Background()); err == nil {
		t.Fatal("expected database readiness error")
	}
	if strings.Join(calls, ",") != "db" {
		t.Fatalf("redis should not be called after database failure: %v", calls)
	}
}

func TestRedisReadinessFailureIsSanitized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	internalError := "redis password=super-secret unavailable"
	r := gin.New()
	registerTopLevelHealth(r, time.Second, newReadinessCheck(
		func(context.Context) error { return nil },
		func(context.Context) error { return errors.New(internalError) },
	))

	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), internalError) || strings.Contains(response.Body.String(), "super-secret") {
		t.Fatalf("readiness response leaked redis error: %s", response.Body.String())
	}
}

type closeRecorder struct {
	calls *[]string
	name  string
}

func (c closeRecorder) Close() error {
	*c.calls = append(*c.calls, c.name)
	return nil
}

func TestCloseDependenciesClosesRedisAndDatabase(t *testing.T) {
	var calls []string
	closeDependencies(closeRecorder{calls: &calls, name: "db"}, func() error {
		calls = append(calls, "redis")
		return nil
	})
	if strings.Join(calls, ",") != "redis,db" {
		t.Fatalf("unexpected cleanup calls: %v", calls)
	}
}

func TestTopLevelReadinessSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerTopLevelHealth(r, time.Second, func(context.Context) error { return nil })

	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("readiness status = %d", response.Code)
	}
}
