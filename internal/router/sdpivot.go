package router

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/middleware"
)

// SDPivotRouterParams holds dependencies for the SDPivot router.
// Uses dig.In so all fields are resolved from the DI container.
type SDPivotRouterParams struct {
	dig.In

	DB          *gorm.DB
	RedisClient *redis.Client
}

// SDPivotRouter holds all SDPivot-specific route handlers.
type SDPivotRouter struct {
	db         *gorm.DB
	redis      *redis.Client
	jwtManager *auth.JWTManager
}

// NewSDPivotRouter creates a new SDPivot router via DI.
// The JWT secret is read from SDP_JWT_SECRET, with SMARTKNORA_JWT_SECRET as a compatibility fallback.
func NewSDPivotRouter(params SDPivotRouterParams) *SDPivotRouter {
	jwtSecret := getSDPivotEnv("SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET")
	if jwtSecret == "" {
		log.Println("WARNING: Using default JWT secret. Set SDP_JWT_SECRET in production!")
		jwtSecret = "sdp-dev-secret-change-in-production"
	}

	jwtConfig := auth.DefaultJWTConfig(jwtSecret)
	jwtManager := auth.NewJWTManager(jwtConfig)

	return &SDPivotRouter{
		db:         params.DB,
		redis:      params.RedisClient,
		jwtManager: jwtManager,
	}
}

// RegisterRoutes registers all SDPivot routes on the given engine.
// Routes are mounted under /api/v1/sdp/
func (sr *SDPivotRouter) RegisterRoutes(r *gin.Engine) {
	sr.registerRoutes(r.Group("/api/v1/sdp"))
	sr.registerRoutes(r.Group("/api/v1/smartknora"))
}

func (sr *SDPivotRouter) registerRoutes(sk *gin.RouterGroup) {
	// Create handler instances
	authHandler := handler.NewSDPivotAuthHandler(sr.db, sr.jwtManager, sr.redis)
	orgHandler := handler.NewSDPivotOrgHandler(sr.db)
	spaceHandler := handler.NewSDPivotSpaceHandler(sr.db)
	tokenHandler := handler.NewSDPivotTokenHandler(sr.db)
	opsHandler := handler.NewSDPivotOpsHandler(sr.db, sr.jwtManager)

	// Public routes (no auth required)
	public := sk.Group("")
	{
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "service": "sdp", "version": "2.0.0"})
		})
	}

	// Auth routes (public)
	authHandler.RegisterRoutes(sk)

	// Ops admin auth routes (public)
	opsHandler.RegisterPublicRoutes(sk)

	// Protected routes (require JWT)
	protected := sk.Group("")
	protected.Use(middleware.SDPivotAuth(sr.jwtManager))
	protected.Use(middleware.SDPivotTenantContext(sr.db))
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
		docHandler := handler.NewSDPivotDocumentHandler(sr.db)
		docHandler.RegisterRoutes(protected)

		// AI Q&A + Admin
		qaHandler := handler.NewSDPivotQAHandler(sr.db)
		qaHandler.RegisterRoutes(protected)

		// AI Writing + Operations
		writingHandler := handler.NewSDPivotWritingHandler(sr.db)
		writingHandler.RegisterRoutes(protected)

		// Ops admin protected routes
		opsHandler.RegisterProtectedRoutes(protected)
	}
}

func getSDPivotEnv(primary, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(fallback)
}
