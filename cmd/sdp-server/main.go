// SDPivot (随越·智枢) standalone server
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

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/middleware"
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
	jwtSecret := getEnvFallback("SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET", "")
	if jwtSecret == "" {
		log.Println("WARNING: Using default JWT secret. Set SDP_JWT_SECRET in production!")
		jwtSecret = "sdp-dev-secret-change-in-production"
	}
	port := getEnv("PORT", "8081")

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

	// ── JWT Manager ────────────────────────────────────────────
	jwtConfig := auth.DefaultJWTConfig(jwtSecret)
	jwtManager := auth.NewJWTManager(jwtConfig)

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
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "sdp", "version": "2.0.0"})
	})

	registerSDPivotRoutes := func(sk *gin.RouterGroup) {
		registerSDPivotHealth(sk)

		// Public routes
		authHandler := handler.NewSDPivotAuthHandler(db, jwtManager, redisClient)
		authHandler.RegisterRoutes(sk)

		// Protected routes
		protected := sk.Group("")
		protected.Use(middleware.SDPivotAuth(jwtManager))
		protected.Use(middleware.SDPivotTenantContext(db))
		protected.Use(middleware.TokenMeteringMiddleware(db))
		{
			// Sprint 1: User profile
			protected.GET("/profile", func(c *gin.Context) {
				c.JSON(200, gin.H{"user_id": middleware.GetUserID(c)})
			})
			protected.PUT("/profile", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "profile updated"})
			})
			protected.PUT("/password", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "password changed"})
			})

			// Sprint 1: Organization management
			orgHandler := handler.NewSDPivotOrgHandler(db)
			orgHandler.RegisterRoutes(protected)

			// Sprint 1: Knowledge space management
			spaceHandler := handler.NewSDPivotSpaceHandler(db)
			spaceHandler.RegisterRoutes(protected)

			// Sprint 1: Token usage queries
			tokenHandler := handler.NewSDPivotTokenHandler(db)
			tokenHandler.RegisterRoutes(protected)

			// Sprint 2: Document management
			docHandler := handler.NewSDPivotDocumentHandler(db)
			docHandler.RegisterRoutes(protected)

			// Sprint 3: AI Q&A + Admin
			qaHandler := handler.NewSDPivotQAHandler(db)
			qaHandler.RegisterRoutes(protected)

			// Sprint 4: AI Writing + Operations
			writingHandler := handler.NewSDPivotWritingHandler(db)
			writingHandler.RegisterRoutes(protected)

			// Ops admin auth (PRD 1.1.4)
			opsHandler := handler.NewSDPivotOpsHandler(db, jwtManager)
			opsHandler.RegisterPublicRoutes(sk)
			opsHandler.RegisterProtectedRoutes(protected)

			// Ops admin management handler (PRD 4.1)
			opsAdminHandler := handler.NewSDPivotOpsAdminHandler(db)
			opsAdminHandler.RegisterOpsRoutes(protected)
			opsAdminHandler.RegisterPublicOpsRoutes(sk)
		}
	}

	registerSDPivotRoutes(r.Group("/api/v1/sdp"))
	registerSDPivotRoutes(r.Group("/api/v1/smartknora"))

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

func registerSDPivotHealth(group *gin.RouterGroup) {
	group.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "sdp", "version": "2.0.0"})
	})
}
