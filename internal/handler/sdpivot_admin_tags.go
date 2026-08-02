package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SDPivotAdminHandler exposes SDPivot-only administration compatibility APIs.
type SDPivotAdminHandler struct {
	db *gorm.DB
}

func NewSDPivotAdminHandler(db *gorm.DB) *SDPivotAdminHandler {
	return &SDPivotAdminHandler{db: db}
}

func (h *SDPivotAdminHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionUserRoleAssign))
	admin.GET("/tags", h.ListTagDictionary)
	admin.POST("/tags", h.CreateTagDictionaryEntry)
	admin.PUT("/tags/:id", h.UpdateTagDictionaryEntry)
	admin.DELETE("/tags/:id", h.DeleteTagDictionaryEntry)
}

// ListTagDictionary returns the shared platform classification dictionary.
func (h *SDPivotAdminHandler) ListTagDictionary(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	dimensions, tags, err := repository.NewDocumentTagRepository(db).ListClassificationDictionary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tag dictionary"})
		return
	}
	if dimensions == nil {
		dimensions = make([]*types.TagDimension, 0)
	}
	if tags == nil {
		tags = make([]*types.TagDictionary, 0)
	}
	c.JSON(http.StatusOK, gin.H{"dimensions": dimensions, "tags": tags})
}

func (h *SDPivotAdminHandler) tagSystemHandler(c *gin.Context) *SystemHandler {
	return &SystemHandler{db: middleware.TenantDB(c, h.db)}
}

func (h *SDPivotAdminHandler) CreateTagDictionaryEntry(c *gin.Context) {
	h.tagSystemHandler(c).CreateTagDictionaryEntry(c)
}

func (h *SDPivotAdminHandler) UpdateTagDictionaryEntry(c *gin.Context) {
	h.tagSystemHandler(c).UpdateTagDictionaryEntry(c)
}

func (h *SDPivotAdminHandler) DeleteTagDictionaryEntry(c *gin.Context) {
	h.tagSystemHandler(c).DeleteTagDictionaryEntry(c)
}
