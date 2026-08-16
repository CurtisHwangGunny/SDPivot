package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SDPivotAdminHandler struct {
	db           *gorm.DB
	documentTags interfaces.DocumentTagRepository
}

func NewSDPivotAdminHandler(db *gorm.DB, documentTags interfaces.DocumentTagRepository) *SDPivotAdminHandler {
	return &SDPivotAdminHandler{db: db, documentTags: documentTags}
}
func (h *SDPivotAdminHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/admin/tags/dictionary", h.GetRecommendedTagDictionary)
	rg.GET("/admin/tags/adjustment-log", h.ListTagAdjustmentLog)
	rg.GET("/admin/tags/learning", h.GetTagLearningStatus)
	rg.GET("/admin/tags", h.ListTagDictionary)
	rg.POST("/admin/tags", h.CreateTagDictionaryEntry)
	rg.PUT("/admin/tags/:id", h.UpdateTagDictionaryEntry)
	rg.DELETE("/admin/tags/:id", h.DeleteTagDictionaryEntry)
}
func (h *SDPivotAdminHandler) ListTagDictionary(c *gin.Context) {
	dimensions, tags, err := h.documentTags.ListClassificationDictionary(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to list tag dictionary"})
		return
	}
	c.JSON(200, gin.H{"dimensions": dimensions, "tags": tags})
}
func (h *SDPivotAdminHandler) GetRecommendedTagDictionary(c *gin.Context) { h.ListTagDictionary(c) }

type tagDictionaryRequest struct {
	DimensionID string `json:"dimension_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color"`
	SortOrder   int    `json:"sort_order"`
}

func (h *SDPivotAdminHandler) CreateTagDictionaryEntry(c *gin.Context) {
	var req tagDictionaryRequest
	if c.ShouldBindJSON(&req) != nil || req.Name == "" {
		c.JSON(400, gin.H{"error": "invalid tag"})
		return
	}
	tag := &types.TagDictionary{ID: uuid.NewString(), DimensionID: req.DimensionID, Name: req.Name, Color: req.Color, SortOrder: req.SortOrder, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := h.db.WithContext(c.Request.Context()).Create(tag).Error; err != nil {
		c.JSON(409, gin.H{"error": "failed to create tag"})
		return
	}
	c.JSON(201, tag)
}
func (h *SDPivotAdminHandler) UpdateTagDictionaryEntry(c *gin.Context) {
	var req tagDictionaryRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "invalid tag"})
		return
	}
	var tag types.TagDictionary
	db := h.db.WithContext(c.Request.Context())
	if err := db.First(&tag, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "tag not found"})
		return
	}
	if req.Name != "" {
		tag.Name = req.Name
	}
	tag.DimensionID = req.DimensionID
	tag.Color = req.Color
	tag.SortOrder = req.SortOrder
	tag.UpdatedAt = time.Now()
	if err := db.Save(&tag).Error; err != nil {
		c.JSON(409, gin.H{"error": "failed to update tag"})
		return
	}
	c.JSON(200, tag)
}
func (h *SDPivotAdminHandler) DeleteTagDictionaryEntry(c *gin.Context) {
	result := h.db.WithContext(c.Request.Context()).Delete(&types.TagDictionary{}, "id = ?", c.Param("id"))
	if result.Error != nil {
		c.JSON(409, gin.H{"error": "failed to delete tag"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(404, gin.H{"error": "tag not found"})
		return
	}
	c.Status(204)
}
func (h *SDPivotAdminHandler) ListTagAdjustmentLog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var items []types.SDPivotTagAdjustmentLog
	db := h.db.WithContext(c.Request.Context()).Where("tenant_id = ?", tenantID(c))
	var total int64
	db.Model(&types.SDPivotTagAdjustmentLog{}).Count(&total)
	db.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items)
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "page_size": size})
}
func (h *SDPivotAdminHandler) GetTagLearningStatus(c *gin.Context) {
	var sample, correct int64
	db := h.db.WithContext(c.Request.Context()).Model(&types.SDPivotTagFeedback{}).Where("tenant_id = ?", tenantID(c))
	db.Count(&sample)
	h.db.WithContext(c.Request.Context()).Model(&types.SDPivotTagFeedback{}).Where("tenant_id = ? AND feedback = ?", tenantID(c), "correct").Count(&correct)
	accuracy := float64(0)
	if sample > 0 {
		accuracy = float64(correct) / float64(sample)
	}
	c.JSON(http.StatusOK, gin.H{"enabled": true, "sample_count": sample, "recent_accuracy": accuracy, "trend": []gin.H{}})
}
