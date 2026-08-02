package router

import (
	"crypto/rand"
	"encoding/base64"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/middleware"
)

// SDPivotRouterParams holds dependencies for the SDPivot router.
// Uses dig.In so all fields are resolved from the DI container.
type SDPivotRouterParams struct {
	dig.In

	DB          *gorm.DB
	RedisClient *redis.Client
	Config      *config.Config
}

// SDPivotRouter holds all SDPivot-specific route handlers.
type SDPivotRouter struct {
	db         *gorm.DB
	redis      *redis.Client
	jwtManager *auth.JWTManager
	product    *config.ProductConfig
}

// NewSDPivotRouter creates a new SDPivot router via DI.
// The JWT secret is read from SDP_JWT_SECRET, with SMARTKNORA_JWT_SECRET as a compatibility fallback.
func NewSDPivotRouter(params SDPivotRouterParams) *SDPivotRouter {
	jwtSecret := getSDPivotEnv("SDP_JWT_SECRET", "SMARTKNORA_JWT_SECRET")
	if jwtSecret == "" {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			panic("failed to generate SDPivot JWT secret: " + err.Error())
		}
		jwtSecret = base64.RawURLEncoding.EncodeToString(secret)
	}

	jwtConfig := auth.DefaultJWTConfig(jwtSecret)
	jwtManager := auth.NewJWTManager(jwtConfig)
	product := config.DefaultProductConfig()
	if params.Config != nil && params.Config.Product != nil {
		configured := *params.Config.Product
		product = &configured
		if product.Brand == "" {
			product.Brand = "sdpivot"
		}
	}

	return &SDPivotRouter{
		db:         params.DB,
		redis:      params.RedisClient,
		jwtManager: jwtManager,
		product:    product,
	}
}

// RegisterRoutes registers canonical SDPivot routes and the optional legacy alias.
func (sr *SDPivotRouter) RegisterRoutes(r *gin.Engine) {
	sr.registerRoutes(r.Group("/api/v1/sdp"))
	// Register auth routes under the REST canonical path without extra prefix.
	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/login", handler.NewSDPivotAuthHandler(sr.db, sr.jwtManager, sr.redis).Login)
		authGroup.POST("/refresh", handler.NewSDPivotAuthHandler(sr.db, sr.jwtManager, sr.redis).RefreshToken)
		authGroup.POST("/logout", handler.NewSDPivotAuthHandler(sr.db, sr.jwtManager, sr.redis).Logout)
	}
	if sr.product != nil && sr.product.EnableLegacyAlias {
		sr.registerRoutes(r.Group("/api/v1/smartknora"))
	}
}

func (sr *SDPivotRouter) registerRoutes(sk *gin.RouterGroup) {
	// Create handler instances
	authHandler := handler.NewSDPivotAuthHandler(sr.db, sr.jwtManager, sr.redis)
	orgHandler := handler.NewSDPivotOrgHandler(sr.db)
	spaceHandler := handler.NewSDPivotSpaceHandler(sr.db)
	tokenHandler := handler.NewSDPivotTokenHandler(sr.db)
	opsHandler := handler.NewSDPivotOpsHandler(sr.db, sr.jwtManager, sr.product.OPMode)
	opsAdminHandler := handler.NewSDPivotOpsAdminHandler(sr.db, sr.product.OPMode)

	// Public routes (no auth required)
	public := sk.Group("")
	{
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "service": "sdp", "version": "2.0.0"})
		})
	}

	// Auth routes (public)
	authHandler.RegisterRoutes(sk)

	// Ops admin auth and anonymous announcement routes
	opsHandler.RegisterPublicRoutes(sk)
	opsAdminHandler.RegisterPublicOpsRoutes(sk)
	if sr.product.OPMode {
		orgs := sk.Group("/organizations")
		orgs.Any("", handler.OPFeatureDisabled)
		orgs.Any("/*path", handler.OPFeatureDisabled)
	}

	// Protected routes (require JWT)
	protected := sk.Group("")
	protected.Use(middleware.SDPivotAuth(sr.jwtManager))
	protected.Use(middleware.SDPivotTenantContext(sr.db))
	protected.Use(middleware.TokenMeteringMiddleware(sr.db))
	{
		// User profile
		authHandler.RegisterProtectedRoutes(protected)
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
		if !sr.product.OPMode {
			orgHandler.RegisterRoutes(protected)
		}

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
		opsAdminHandler.RegisterOpsRoutes(protected)
	}
}

func getSDPivotEnv(primary, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(fallback)
}
