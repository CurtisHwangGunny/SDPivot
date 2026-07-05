// smartKnora (随越·智枢) standalone server
// Connects directly to PostgreSQL + Redis, serves smartKnora API
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
	dbHost := getEnv("SMART_DB_HOST", "localhost")
	dbPort := getEnv("SMART_DB_PORT", "5432")
	dbUser := getEnv("SMART_DB_USER", "postgres")
	dbPass := getEnv("SMART_DB_PASSWORD", "postgres")
	dbName := getEnv("SMART_DB_NAME", "WeKnora")
	redisAddr := getEnv("REDIS_ADDR", "")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	jwtSecret := getEnv("SMARTKNORA_JWT_SECRET", "")
	if jwtSecret == "" {
		log.Println("WARNING: Using default JWT secret. Set SMARTKNORA_JWT_SECRET in production!")
		jwtSecret = "smartknora-dev-secret-change-in-production"
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
		c.JSON(200, gin.H{"status": "ok", "service": "smartknora", "version": "2.0.0"})
	})

	// ── smartKnora API ─────────────────────────────────────────
	sk := r.Group("/api/v1/smartknora")

	// Public routes
	authHandler := handler.NewSmartKnoraAuthHandler(db, jwtManager, redisClient)
	authHandler.RegisterRoutes(sk)

	// Protected routes
	protected := sk.Group("")
	protected.Use(middleware.SmartKnoraAuth(jwtManager))
	protected.Use(middleware.SmartKnoraTenantContext(db))
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
		orgHandler := handler.NewSmartKnoraOrgHandler(db)
		orgHandler.RegisterRoutes(protected)

		// Sprint 1: Knowledge space management
		spaceHandler := handler.NewSmartKnoraSpaceHandler(db)
		spaceHandler.RegisterRoutes(protected)

		// Sprint 1: Token usage queries
		tokenHandler := handler.NewSmartKnoraTokenHandler(db)
		tokenHandler.RegisterRoutes(protected)

		// Sprint 2: Document management
		docHandler := handler.NewSmartKnoraDocumentHandler(db)
		docHandler.RegisterRoutes(protected)

		// Sprint 3: AI Q&A + Admin
		qaHandler := handler.NewSmartKnoraQAHandler(db)
		qaHandler.RegisterRoutes(protected)

		// Sprint 4: AI Writing + Operations
		writingHandler := handler.NewSmartKnoraWritingHandler(db)
		writingHandler.RegisterRoutes(protected)

		// Ops admin auth (PRD 1.1.4)
		opsHandler := handler.NewSmartKnoraOpsHandler(db, jwtManager)
		opsHandler.RegisterPublicRoutes(sk)
		opsHandler.RegisterProtectedRoutes(protected)

		// Ops admin management handler (PRD 4.1)
		opsAdminHandler := handler.NewSmartKnoraOpsAdminHandler(db)
		opsAdminHandler.RegisterOpsRoutes(protected)
		opsAdminHandler.RegisterPublicOpsRoutes(sk)
	}

	// ── Start Server ───────────────────────────────────────────
	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("[Server] smartKnora v2.0.0 starting on :%s", port)
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
	value := os.Getenv("SMARTKNORA_ALLOWED_ORIGINS")
	if value == "" {
		return []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://43.133.61.77:3099",
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
