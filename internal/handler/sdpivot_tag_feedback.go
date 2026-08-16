package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SDPivotTagFeedbackHandler struct{ db *gorm.DB }

func NewSDPivotTagFeedbackHandler(db *gorm.DB) *SDPivotTagFeedbackHandler {
	return &SDPivotTagFeedbackHandler{db: db}
}
func (h *SDPivotTagFeedbackHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.PUT("/knowledge/:id/tags", h.UpdateDocumentTags)
	rg.POST("/knowledge/batch-tag", h.BatchTag)
	rg.POST("/tags/feedback", h.CreateFeedback)
	rg.GET("/admin/tag-feedback/queue", h.ListFeedbackQueue)
	rg.POST("/admin/tag-feedback/review", h.ReviewFeedback)
}

type updateDocumentTagsRequest struct {
	TagIDs []string `json:"tag_ids" binding:"required"`
	Reason string   `json:"reason"`
}

func (h *SDPivotTagFeedbackHandler) UpdateDocumentTags(c *gin.Context) {
	var req updateDocumentTagsRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "invalid tag request"})
		return
	}
	if err := h.replaceTags(c, c.Param("id"), uniqueStrings(req.TagIDs), req.Reason); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"document_id": c.Param("id"), "tag_count": len(uniqueStrings(req.TagIDs))})
}

type batchTagRequest struct {
	DocumentIDs []string `json:"document_ids" binding:"required"`
	TagID       string   `json:"tag_id" binding:"required"`
	Action      string   `json:"action" binding:"required"`
	Reason      string   `json:"reason"`
}

func (h *SDPivotTagFeedbackHandler) BatchTag(c *gin.Context) {
	var req batchTagRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "invalid batch tag request"})
		return
	}
	if req.Action != "apply" && req.Action != "replace" {
		c.JSON(400, gin.H{"error": "action must be apply or replace"})
		return
	}
	updated := 0
	for _, id := range uniqueStrings(req.DocumentIDs) {
		ids := []string{req.TagID}
		if req.Action == "apply" {
			var existing []types.SDPivotDocumentTag
			if err := h.db.WithContext(c.Request.Context()).Where("tenant_id = ? AND document_id = ? AND deleted_at IS NULL", tenantID(c), id).Find(&existing).Error; err != nil {
				continue
			}
			for _, row := range existing {
				ids = append(ids, row.TagID)
			}
		}
		if h.replaceTags(c, id, uniqueStrings(ids), req.Reason) == nil {
			updated++
		}
	}
	c.JSON(200, gin.H{"updated": updated})
}
func (h *SDPivotTagFeedbackHandler) replaceTags(c *gin.Context, documentID string, tagIDs []string, reason string) error {
	var knowledge types.Knowledge
	if err := h.db.WithContext(c.Request.Context()).Where("id = ? AND tenant_id = ?", documentID, tenantID(c)).First(&knowledge).Error; err != nil {
		return err
	}
	var refs []struct {
		ID          string
		DimensionID string
	}
	if len(tagIDs) > 0 {
		if err := h.db.WithContext(c.Request.Context()).Table("tag_dictionary AS dict").Select("dict.id, dict.dimension_id").Joins("JOIN tag_dimensions AS dim ON dim.id = dict.dimension_id").Where("dict.id IN ? AND dim.deleted_at IS NULL AND dim.enabled = TRUE AND (dim.tenant_id IS NULL OR dim.tenant_id = ?)", tagIDs, tenantID(c)).Scan(&refs).Error; err != nil {
			return err
		}
		if len(refs) != len(tagIDs) {
			return gorm.ErrRecordNotFound
		}
	}
	return h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var old []types.SDPivotDocumentTag
		if err := tx.Where("tenant_id = ? AND document_id = ? AND deleted_at IS NULL", tenantID(c), documentID).Find(&old).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND document_id = ?", tenantID(c), documentID).Delete(&types.SDPivotDocumentTag{}).Error; err != nil {
			return err
		}
		now := time.Now()
		for _, ref := range refs {
			if err := tx.Create(&types.SDPivotDocumentTag{ID: uuid.NewString(), TenantID: tenantID(c), DocumentID: documentID, DimensionID: ref.DimensionID, TagID: ref.ID, Source: "manual", Confidence: 1, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
				return err
			}
		}
		_ = reason
		return nil
	})
}

type tagFeedbackRequest struct {
	DocumentID string `json:"document_id" binding:"required"`
	TagID      string `json:"tag_id" binding:"required"`
	Feedback   string `json:"feedback" binding:"required"`
}

func (h *SDPivotTagFeedbackHandler) CreateFeedback(c *gin.Context) {
	var req tagFeedbackRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "invalid feedback"})
		return
	}
	req.Feedback = strings.ToLower(strings.TrimSpace(req.Feedback))
	if req.Feedback != "correct" && req.Feedback != "incorrect" {
		c.JSON(400, gin.H{"error": "feedback must be correct or incorrect"})
		return
	}
	var tag types.TagDictionary
	if err := h.db.WithContext(c.Request.Context()).First(&tag, "id = ?", req.TagID).Error; err != nil {
		c.JSON(400, gin.H{"error": "tag not found"})
		return
	}
	now := time.Now()
	row := types.SDPivotTagFeedback{ID: uuid.NewString(), TenantID: tenantID(c), DocumentID: req.DocumentID, TagID: req.TagID, OriginalTag: tag.Name, Feedback: req.Feedback, Status: "pending", CreatedBy: userID(c), CreatedAt: now, UpdatedAt: now}
	if err := h.db.WithContext(c.Request.Context()).Create(&row).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to create tag feedback"})
		return
	}
	c.JSON(201, row)
}
func (h *SDPivotTagFeedbackHandler) ListFeedbackQueue(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	var rows []types.SDPivotTagFeedback
	if err := h.db.WithContext(c.Request.Context()).Where("tenant_id = ? AND status = ?", tenantID(c), status).Order("created_at ASC").Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to list tag feedback queue"})
		return
	}
	c.JSON(200, gin.H{"items": rows, "total": len(rows)})
}

type reviewFeedbackRequest struct {
	IDs      []string `json:"ids" binding:"required"`
	Decision string   `json:"decision" binding:"required"`
}

func (h *SDPivotTagFeedbackHandler) ReviewFeedback(c *gin.Context) {
	var req reviewFeedbackRequest
	if c.ShouldBindJSON(&req) != nil || (req.Decision != "reviewed" && req.Decision != "rejected") {
		c.JSON(400, gin.H{"error": "invalid review request"})
		return
	}
	now := time.Now()
	result := h.db.WithContext(c.Request.Context()).Model(&types.SDPivotTagFeedback{}).Where("tenant_id = ? AND status = ? AND id IN ?", tenantID(c), "pending", uniqueStrings(req.IDs)).Updates(map[string]any{"status": req.Decision, "reviewed_by": userID(c), "reviewed_at": now, "updated_at": now})
	if result.Error != nil {
		c.JSON(500, gin.H{"error": "failed to review tag feedback"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reviewed": result.RowsAffected, "status": req.Decision})
}
func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
