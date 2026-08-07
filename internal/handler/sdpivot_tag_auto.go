package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

type SDPivotTagAutoHandler struct {
	db *gorm.DB
}

func NewSDPivotTagAutoHandler(db *gorm.DB) *SDPivotTagAutoHandler {
	return &SDPivotTagAutoHandler{db: db}
}

func (h *SDPivotTagAutoHandler) RegisterRoutes(rg *gin.RouterGroup) {
	docs := rg.Group("/documents", middleware.RequirePermission(middleware.PermissionKnowledgeWrite))
	docs.POST("/:id/auto-tag", h.AutoTag)
}

type autoTagCandidate struct {
	TagID       string
	DimensionID string
	Name        string
}

func (h *SDPivotTagAutoHandler) AutoTag(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	doc, ok := authorizeTagDocument(c, db, c.Param("id"), spaceAccessEdit)
	if !ok {
		return
	}
	var chunks []types.SDPivotDocumentChunk
	if err := db.Where("tenant_id = ? AND document_id = ?", doc.TenantID, doc.ID).Order("chunk_index ASC").Find(&chunks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load document content"})
		return
	}
	var text strings.Builder
	text.WriteString(doc.Title)
	for _, chunk := range chunks {
		text.WriteByte('\n')
		text.WriteString(chunk.Content)
		if text.Len() >= 200000 {
			break
		}
	}
	content := strings.ToLower(text.String())
	var candidates []autoTagCandidate
	if err := db.Table("tag_dictionary AS dict").
		Select("dict.id AS tag_id, dict.dimension_id, dict.name").
		Joins("JOIN tag_dimensions AS dim ON dim.id = dict.dimension_id").
		Where("dim.deleted_at IS NULL AND dim.enabled = TRUE AND (dim.tenant_id IS NULL OR dim.tenant_id = ?)", doc.TenantID).
		Order("dim.sort_order ASC, dict.sort_order ASC").Scan(&candidates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tag dictionary"})
		return
	}
	created := make([]types.SDPivotDocumentTag, 0)
	now := time.Now()
	for _, candidate := range candidates {
		keyword := strings.ToLower(strings.TrimSpace(candidate.Name))
		if len([]rune(keyword)) < 2 || !strings.Contains(content, keyword) {
			continue
		}
		assignment := types.SDPivotDocumentTag{ID: uuid.NewString(), TenantID: doc.TenantID, DocumentID: doc.ID, DimensionID: candidate.DimensionID, TagID: candidate.TagID, Source: "auto", Confidence: 0.85, CreatedAt: now, UpdatedAt: now}
		var existing types.SDPivotDocumentTag
		err := db.Unscoped().Where("tenant_id = ? AND document_id = ? AND tag_id = ?", doc.TenantID, doc.ID, candidate.TagID).First(&existing).Error
		if err == nil {
			if existing.DeletedAt.Valid {
				if updateErr := db.Unscoped().Model(&existing).Updates(map[string]any{"dimension_id": candidate.DimensionID, "source": "auto", "confidence": 0.85, "updated_at": now, "deleted_at": nil}).Error; updateErr == nil {
					created = append(created, assignment)
				}
			}
			continue
		}
		if err != gorm.ErrRecordNotFound {
			continue
		}
		if err := db.Create(&assignment).Error; err == nil {
			created = append(created, assignment)
		}
	}
	c.JSON(http.StatusOK, gin.H{"document_id": doc.ID, "created": len(created), "tags": created})
}
