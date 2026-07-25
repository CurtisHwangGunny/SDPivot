package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTokenUsagePerformanceRouter(tb testing.TB, rowCount int) *gin.Engine {
	tb.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		tb.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		tb.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	tb.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&types.SDPivotTokenUsage{}); err != nil {
		tb.Fatal(err)
	}

	createdAt := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	tx := db.Begin()
	if tx.Error != nil {
		tb.Fatal(tx.Error)
	}
	for i := 0; i < rowCount; i++ {
		userID := "performance-user"
		tenantID := uint64(7)
		if i%4 == 0 {
			userID = "other-user"
		}
		if i%5 == 0 {
			tenantID = 8
		}
		if err := tx.Exec(`
			INSERT INTO token_usage
				(id, user_id, tenant_id, model, input_tokens, output_tokens, action, created_at)
			VALUES (?, ?, ?, 'performance-model', 100, 50, '/qa', ?)
		`, fmt.Sprintf("usage-%06d", i), userID, tenantID, createdAt.Add(time.Duration(i)*time.Second)).Error; err != nil {
			tb.Fatal(err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		tb.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "performance-user")
		c.Set("tenant_id", uint64(7))
		c.Next()
	})
	NewSDPivotTokenHandler(db).RegisterRoutes(router.Group(""))
	return router
}

func BenchmarkTokenUsageAggregation(b *testing.B) {
	router := setupTokenUsagePerformanceRouter(b, 10000)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		request := httptest.NewRequest(http.MethodGet, "/usage/summary", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatalf("status = %d", response.Code)
		}
	}
}
