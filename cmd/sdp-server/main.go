// SDPivot standalone server
// Connects directly to PostgreSQL + Redis, serves SDPivot API
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/router"
)

func main() {
	// ── Config from env ────────────────────────────────────────
	dbHost := getEnvFallback("SDP_DB_HOST", "SMART_DB_HOST", "localhost")
	dbPort := getEnvFallback("SDP_DB_PORT", "SMART_DB_PORT", "5432")
	dbUser := getEnvFallback("SDP_DB_USER", "SMART_DB_USER", "postgres")
	dbPass := getEnvFallback("SDP_DB_PASSWORD", "SMART_DB_PASSWORD", "postgres")
	dbName := getEnvFallback("SDP_DB_NAME", "SMART_DB_NAME", "WeKnora")
	redisAddr := getEnvFallback("SDP_REDIS_ADDR", "REDIS_ADDR", "")
	redisPassword := getEnvFallback("SDP_REDIS_PASSWORD", "REDIS_PASSWORD", "")
	port := getEnv("PORT", "8081")
	product := config.DefaultProductConfig()
	if err := config.ApplyProductEnvOverrides(product); err != nil {
		log.Fatalf("Invalid product configuration: %v", err)
	}

	// ── Database ───────────────────────────────────────────────
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		dbHost, dbPort, dbUser, dbPass, dbName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	log.Printf("[DB] Connected to PostgreSQL %s:%s/%s", dbHost, dbPort, dbName)

	// ── Redis (optional) ───────────────────────────────────────
	var redisClient *redis.Client
	if redisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: redisAddr, Password: redisPassword})
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			log.Printf("[Redis] Warning: %v", err)
		} else {
			log.Printf("[Redis] Connected to %s", redisAddr)
		}
	}

	// ── Gin Router ─────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     getAllowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Health check ───────────────────────────────────────────
	registerTopLevelHealth(r)

	router.NewSDPivotRouter(router.SDPivotRouterParams{
		DB:          db,
		RedisClient: redisClient,
		Config:      &config.Config{Product: product},
	}).RegisterRoutes(r)

	// ── Start Server ───────────────────────────────────────────
	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("[Server] SDPivot v2.0.0 starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[Server] Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("[Server] Stopped")
}

func registerTopLevelHealth(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "sdp", "version": "2.0.0"})
	})
}

func getAllowedOrigins() []string {
	value := getEnvFallback("SDP_ALLOWED_ORIGINS", "SMARTKNORA_ALLOWED_ORIGINS", "")
	if value == "" {
		return []string{
			// Dev server (43.133.61.77)
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://43.133.61.77:3099",
			// Test server (47.110.51.90)
			"http://47.110.51.90:3099",
			"http://localhost:3099",
			"http://127.0.0.1:3099",
		}
	}

	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvFallback(primary, fallback, def string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return getEnv(fallback, def)
}
