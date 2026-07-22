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
	"strconv"
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
	if err := validateStandaloneConfig(product); err != nil {
		log.Fatalf("Invalid standalone configuration: %v", err)
	}
	poolConfig, err := loadDBPoolConfig()
	if err != nil {
		log.Fatalf("Invalid database pool configuration: %v", err)
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
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("[DB] Close warning: %v", err)
		}
	}()
	log.Printf("[DB] Connected to PostgreSQL %s:%s/%s", dbHost, dbPort, dbName)

	// ── Redis (optional) ───────────────────────────────────────
	var redisClient *redis.Client
	if redisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: redisAddr, Password: redisPassword})
		defer func() {
			if err := redisClient.Close(); err != nil {
				log.Printf("[Redis] Close warning: %v", err)
			}
		}()
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

	// ── Health checks ──────────────────────────────────────────
	registerTopLevelHealth(r, func(ctx context.Context) error {
		if err := sqlDB.PingContext(ctx); err != nil {
			return err
		}
		if redisClient != nil {
			return redisClient.Ping(ctx).Err()
		}
		return nil
	})

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

type dbPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func validateStandaloneConfig(product *config.ProductConfig) error {
	if product == nil || !product.OPMode {
		return nil
	}
	if strings.TrimSpace(getEnvFallback("SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET", "")) == "" {
		return fmt.Errorf("OP mode requires SDP_JWT_SECRET or SMARTKNORA_JWT_SECRET")
	}
	if strings.TrimSpace(getEnvFallback("SDP_DB_PASSWORD", "SMART_DB_PASSWORD", "")) == "" {
		return fmt.Errorf("OP mode requires SDP_DB_PASSWORD or SMART_DB_PASSWORD")
	}
	return nil
}

func loadDBPoolConfig() (dbPoolConfig, error) {
	maxOpen, err := parseEnvInt("SDP_DB_MAX_OPEN_CONNS", 10, false)
	if err != nil {
		return dbPoolConfig{}, err
	}
	maxIdle, err := parseEnvInt("SDP_DB_MAX_IDLE_CONNS", 5, true)
	if err != nil {
		return dbPoolConfig{}, err
	}
	if maxIdle > maxOpen {
		return dbPoolConfig{}, fmt.Errorf("SDP_DB_MAX_IDLE_CONNS must not exceed SDP_DB_MAX_OPEN_CONNS")
	}
	lifetimeValue := strings.TrimSpace(getEnv("SDP_DB_CONN_MAX_LIFETIME", "10m"))
	lifetime, err := time.ParseDuration(lifetimeValue)
	if err != nil || lifetime <= 0 {
		return dbPoolConfig{}, fmt.Errorf("SDP_DB_CONN_MAX_LIFETIME must be a positive duration")
	}
	return dbPoolConfig{MaxOpenConns: maxOpen, MaxIdleConns: maxIdle, ConnMaxLifetime: lifetime}, nil
}

func parseEnvInt(name string, defaultValue int, allowZero bool) (int, error) {
	value := strings.TrimSpace(getEnv(name, strconv.Itoa(defaultValue)))
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 || (!allowZero && parsed == 0) {
		if allowZero {
			return 0, fmt.Errorf("%s must be an integer greater than or equal to zero", name)
		}
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}

func registerTopLevelHealth(r *gin.Engine, readinessCheck func(context.Context) error) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "sdp", "version": "2.0.0"})
	})
	r.GET("/ready", func(c *gin.Context) {
		if readinessCheck != nil && readinessCheck(c.Request.Context()) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
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
