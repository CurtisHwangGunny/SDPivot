package handler

import (
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SDPivotTagAutoHandler struct{ db *gorm.DB }

func NewSDPivotTagAutoHandler(db *gorm.DB) *SDPivotTagAutoHandler {
	return &SDPivotTagAutoHandler{db: db}
}
func (h *SDPivotTagAutoHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/knowledge/:id/auto-tag", h.AutoTag)
}
func (h *SDPivotTagAutoHandler) AutoTag(c *gin.Context) {
	var knowledge types.Knowledge
	db := h.db.WithContext(c.Request.Context())
	if err := db.Where("id = ? AND tenant_id = ?", c.Param("id"), tenantID(c)).First(&knowledge).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge not found"})
		return
	}
	var chunks []*types.Chunk
	if err := db.Where("tenant_id = ? AND knowledge_id = ?", knowledge.TenantID, knowledge.ID).Order("chunk_index ASC").Find(&chunks).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to load document content"})
		return
	}
	var text strings.Builder
	text.WriteString(knowledge.Title)
	for _, chunk := range chunks {
		if chunk != nil {
			text.WriteByte('\n')
			text.WriteString(chunk.Content)
			if text.Len() >= 200000 {
				break
			}
		}
	}
	var candidates []struct {
		TagID       string `gorm:"column:tag_id"`
		DimensionID string `gorm:"column:dimension_id"`
		Name        string
	}
	if err := db.Table("tag_dictionary AS dict").Select("dict.id AS tag_id, dict.dimension_id, dict.name").Joins("JOIN tag_dimensions AS dim ON dim.id = dict.dimension_id").Where("dim.deleted_at IS NULL AND dim.enabled = TRUE AND (dim.tenant_id IS NULL OR dim.tenant_id = ?)", knowledge.TenantID).Order("dim.sort_order ASC, dict.sort_order ASC").Scan(&candidates).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to load tag dictionary"})
		return
	}
	created := 0
	content := strings.ToLower(text.String())
	for _, candidate := range candidates {
		keyword := strings.ToLower(strings.TrimSpace(candidate.Name))
		if len([]rune(keyword)) < 2 || !strings.Contains(content, keyword) {
			continue
		}
		result := db.Exec("INSERT INTO document_tags (tenant_id, document_id, tag_id, dimension_id, confidence, source, updated_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP) ON CONFLICT (document_id, tag_id) WHERE deleted_at IS NULL DO UPDATE SET confidence = EXCLUDED.confidence, source = EXCLUDED.source, updated_at = CURRENT_TIMESTAMP", knowledge.TenantID, knowledge.ID, candidate.TagID, candidate.DimensionID, 0.85, "auto")
		if result.Error == nil {
			created++
		}
	}
	c.JSON(200, gin.H{"document_id": knowledge.ID, "created": created})
}
