package router

import (
	"os"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/middleware"
)

// SmartKnoraRouterParams holds dependencies for the smartKnora router.
// Uses dig.In so all fields are resolved from the DI container.
type SmartKnoraRouterParams struct {
	dig.In

	DB          *gorm.DB
	RedisClient *redis.Client
}

// SmartKnoraRouter holds all smartKnora-specific route handlers.
type SmartKnoraRouter struct {
	db         *gorm.DB
	redis      *redis.Client
	jwtManager *auth.JWTManager
}

// NewSmartKnoraRouter creates a new smartKnora router via DI.
// The JWT secret is read from SMARTKNORA_JWT_SECRET env var.
func NewSmartKnoraRouter(params SmartKnoraRouterParams) *SmartKnoraRouter {
	jwtSecret := os.Getenv("SMARTKNORA_JWT_SECRET")
	if jwtSecret == "" {
		log.Println("WARNING: Using default JWT secret. Set SMARTKNORA_JWT_SECRET in production!")
		jwtSecret = "smartknora-dev-secret-change-in-production"
	}

	jwtConfig := auth.DefaultJWTConfig(jwtSecret)
	jwtManager := auth.NewJWTManager(jwtConfig)

	return &SmartKnoraRouter{
		db:         params.DB,
		redis:      params.RedisClient,
		jwtManager: jwtManager,
	}
}

// RegisterRoutes registers all smartKnora routes on the given engine.
// Routes are mounted under /api/v1/smartknora/
func (sr *SmartKnoraRouter) RegisterRoutes(r *gin.Engine) {
	// Create handler instances
	authHandler := handler.NewSmartKnoraAuthHandler(sr.db, sr.jwtManager, sr.redis)
	orgHandler := handler.NewSmartKnoraOrgHandler(sr.db)
	spaceHandler := handler.NewSmartKnoraSpaceHandler(sr.db)
	tokenHandler := handler.NewSmartKnoraTokenHandler(sr.db)
	opsHandler := handler.NewSmartKnoraOpsHandler(sr.db, sr.jwtManager)

	// Main smartKnora API group
	sk := r.Group("/api/v1/smartknora")

	// Public routes (no auth required)
	public := sk.Group("")
	{
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "service": "smartknora"})
		})
	}

	// Auth routes (public)
	authHandler.RegisterRoutes(sk)

	// Ops admin auth routes (public)
	opsHandler.RegisterPublicRoutes(sk)

	// Protected routes (require JWT)
	protected := sk.Group("")
	protected.Use(middleware.SmartKnoraAuth(sr.jwtManager))
	protected.Use(middleware.SmartKnoraTenantContext(sr.db))
	protected.Use(middleware.TokenMeteringMiddleware(sr.db))
	{
		// User profile
		protected.GET("/profile", func(c *gin.Context) {
			c.JSON(200, gin.H{"user_id": middleware.GetUserID(c)})
		})
		protected.PUT("/profile", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "profile updated"})
		})
		protected.PUT("/password", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "password changed"})
		})

		// Organization management
		orgHandler.RegisterRoutes(protected)

		// Knowledge space management
		spaceHandler.RegisterRoutes(protected)

		// Token usage queries
		tokenHandler.RegisterRoutes(protected)

		// Document management
		docHandler := handler.NewSmartKnoraDocumentHandler(sr.db)
		docHandler.RegisterRoutes(protected)

		// AI Q&A + Admin
		qaHandler := handler.NewSmartKnoraQAHandler(sr.db)
		qaHandler.RegisterRoutes(protected)

		// AI Writing + Operations
		writingHandler := handler.NewSmartKnoraWritingHandler(sr.db)
		writingHandler.RegisterRoutes(protected)

		// Ops admin protected routes
		opsHandler.RegisterProtectedRoutes(protected)
	}
}
