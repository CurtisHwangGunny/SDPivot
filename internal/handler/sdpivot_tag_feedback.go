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

type SDPivotTagFeedbackHandler struct {
	db *gorm.DB
}

func NewSDPivotTagFeedbackHandler(db *gorm.DB) *SDPivotTagFeedbackHandler {
	return &SDPivotTagFeedbackHandler{db: db}
}

func (h *SDPivotTagFeedbackHandler) RegisterRoutes(rg *gin.RouterGroup) {
	docs := rg.Group("/documents", middleware.RequirePermission(middleware.PermissionKnowledgeWrite))
	docs.PUT("/:id/tags", h.UpdateDocumentTags)
	docs.POST("/batch-tag", h.BatchTag)

	tags := rg.Group("/tags", middleware.RequirePermission(middleware.PermissionKnowledgeWrite))
	tags.POST("/feedback", h.CreateFeedback)

	admin := rg.Group("/admin/tag-feedback", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	admin.GET("/queue", h.ListFeedbackQueue)
	admin.POST("/review", h.ReviewFeedback)
}

type updateDocumentTagsRequest struct {
	TagIDs []string `json:"tag_ids" binding:"required"`
	Reason string   `json:"reason"`
}

type tagDictionaryRef struct {
	ID          string
	DimensionID string
}

func (h *SDPivotTagFeedbackHandler) UpdateDocumentTags(c *gin.Context) {
	var req updateDocumentTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db := middleware.TenantDB(c, h.db)
	doc, ok := authorizeTagDocument(c, db, c.Param("id"), spaceAccessEdit)
	if !ok {
		return
	}
	tags, err := loadTagDictionaryRefs(db, doc.TenantID, req.TagIDs)
	if err != nil || len(tags) != len(uniqueStrings(req.TagIDs)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "one or more tags are invalid"})
		return
	}
	if err := replaceSDPivotDocumentTags(db, doc, tags, middleware.GetUserID(c), strings.TrimSpace(req.Reason)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document tags"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"document_id": doc.ID, "tag_count": len(tags)})
}

type batchTagRequest struct {
	DocumentIDs []string `json:"document_ids" binding:"required"`
	TagID       string   `json:"tag_id" binding:"required"`
	Action      string   `json:"action" binding:"required"`
	Reason      string   `json:"reason"`
}

func (h *SDPivotTagFeedbackHandler) BatchTag(c *gin.Context) {
	var req batchTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	if req.Action != "apply" && req.Action != "replace" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be apply or replace"})
		return
	}
	documentIDs := uniqueStrings(req.DocumentIDs)
	if len(documentIDs) == 0 || len(documentIDs) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document_ids must contain between 1 and 200 items"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	tagRefs, err := loadTagDictionaryRefs(db, tenantID, []string{req.TagID})
	if err != nil || len(tagRefs) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tag not found"})
		return
	}
	updated := 0
	for _, documentID := range documentIDs {
		doc, ok := authorizeTagDocument(c, db, documentID, spaceAccessEdit)
		if !ok {
			return
		}
		refs := tagRefs
		if req.Action == "apply" {
			var existing []types.SDPivotDocumentTag
			if err := db.Where("tenant_id = ? AND document_id = ?", tenantID, doc.ID).Find(&existing).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load document tags"})
				return
			}
			ids := make([]string, 0, len(existing)+1)
			for _, item := range existing {
				ids = append(ids, item.TagID)
			}
			ids = append(ids, req.TagID)
			refs, err = loadTagDictionaryRefs(db, tenantID, ids)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate document tags"})
				return
			}
		}
		if err := replaceSDPivotDocumentTags(db, doc, refs, middleware.GetUserID(c), strings.TrimSpace(req.Reason)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to batch update document tags"})
			return
		}
		updated++
	}
	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

type tagFeedbackRequest struct {
	DocumentID string `json:"document_id" binding:"required"`
	TagID      string `json:"tag_id" binding:"required"`
	Feedback   string `json:"feedback" binding:"required"`
}

func (h *SDPivotTagFeedbackHandler) CreateFeedback(c *gin.Context) {
	var req tagFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Feedback = strings.ToLower(strings.TrimSpace(req.Feedback))
	if req.Feedback != "correct" && req.Feedback != "incorrect" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "feedback must be correct or incorrect"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	doc, ok := authorizeTagDocument(c, db, req.DocumentID, spaceAccessEdit)
	if !ok {
		return
	}
	var tag types.TagDictionary
	if err := db.Joins("JOIN tag_dimensions AS dim ON dim.id = tag_dictionary.dimension_id").Where("tag_dictionary.id = ? AND dim.deleted_at IS NULL AND (dim.tenant_id IS NULL OR dim.tenant_id = ?)", req.TagID, doc.TenantID).First(&tag).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tag not found"})
		return
	}
	now := time.Now()
	feedback := types.SDPivotTagFeedback{ID: uuid.NewString(), TenantID: doc.TenantID, DocumentID: doc.ID, TagID: tag.ID, OriginalTag: tag.Name, Feedback: req.Feedback, Status: "pending", CreatedBy: middleware.GetUserID(c), CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&feedback).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tag feedback"})
		return
	}
	c.JSON(http.StatusCreated, feedback)
}

type feedbackQueueItem struct {
	types.SDPivotTagFeedback
	DocumentTitle string `json:"document_title"`
	TagName       string `json:"tag_name"`
}

func (h *SDPivotTagFeedbackHandler) ListFeedbackQueue(c *gin.Context) {
	status := strings.TrimSpace(c.DefaultQuery("status", "pending"))
	if status != "pending" && status != "reviewed" && status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback status"})
		return
	}
	var items []feedbackQueueItem
	err := middleware.TenantDB(c, h.db).Table("tag_feedback_queue AS feedback").
		Select("feedback.*, documents.title AS document_title, tag_dictionary.name AS tag_name").
		Joins("JOIN documents ON documents.id = feedback.document_id AND documents.tenant_id = feedback.tenant_id AND documents.deleted_at IS NULL").
		Joins("JOIN tag_dictionary ON tag_dictionary.id = feedback.tag_id").
		Where("feedback.tenant_id = ? AND feedback.status = ?", middleware.GetTenantID(c), status).
		Order("feedback.created_at ASC").Scan(&items).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tag feedback queue"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

type reviewFeedbackRequest struct {
	IDs      []string `json:"ids" binding:"required"`
	Decision string   `json:"decision" binding:"required"`
}

func (h *SDPivotTagFeedbackHandler) ReviewFeedback(c *gin.Context) {
	var req reviewFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ids := uniqueStrings(req.IDs)
	req.Decision = strings.ToLower(strings.TrimSpace(req.Decision))
	if len(ids) == 0 || (req.Decision != "reviewed" && req.Decision != "rejected") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decision must be reviewed or rejected"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	var feedback []types.SDPivotTagFeedback
	if err := db.Where("tenant_id = ? AND status = ? AND id IN ?", tenantID, "pending", ids).Find(&feedback).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load feedback"})
		return
	}
	if len(feedback) != len(ids) {
		c.JSON(http.StatusConflict, gin.H{"error": "one or more feedback items are no longer pending"})
		return
	}
	now := time.Now()
	reviewer := middleware.GetUserID(c)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.SDPivotTagFeedback{}).Where("tenant_id = ? AND id IN ?", tenantID, ids).Updates(map[string]any{"status": req.Decision, "reviewed_by": reviewer, "reviewed_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		if req.Decision == "reviewed" {
			for _, item := range feedback {
				if err := tx.Model(&types.SDPivotTagAdjustmentLog{}).Where("tenant_id = ? AND document_id = ? AND (old_tag_id = ? OR new_tag_id = ?)", tenantID, item.DocumentID, item.TagID, item.TagID).Update("status", "reviewed").Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to review tag feedback"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reviewed": len(ids), "status": req.Decision})
}

func authorizeTagDocument(c *gin.Context, db *gorm.DB, documentID string, level spaceAccessLevel) (*types.SDPivotDocument, bool) {
	var doc types.SDPivotDocument
	if err := db.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", documentID, middleware.GetTenantID(c)).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return nil, false
	}
	if _, ok := authorizeSpace(c, db, doc.SpaceID, level); !ok {
		return nil, false
	}
	return &doc, true
}

func loadTagDictionaryRefs(db *gorm.DB, tenantID uint64, ids []string) ([]tagDictionaryRef, error) {
	ids = uniqueStrings(ids)
	if len(ids) == 0 {
		return []tagDictionaryRef{}, nil
	}
	var refs []tagDictionaryRef
	err := db.Table("tag_dictionary AS dict").Select("dict.id, dict.dimension_id").
		Joins("JOIN tag_dimensions AS dim ON dim.id = dict.dimension_id").
		Where("dict.id IN ? AND dim.deleted_at IS NULL AND dim.enabled = TRUE AND (dim.tenant_id IS NULL OR dim.tenant_id = ?)", ids, tenantID).
		Scan(&refs).Error
	return refs, err
}

func replaceSDPivotDocumentTags(db *gorm.DB, doc *types.SDPivotDocument, refs []tagDictionaryRef, operator, reason string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var existing []types.SDPivotDocumentTag
		if err := tx.Where("tenant_id = ? AND document_id = ?", doc.TenantID, doc.ID).Find(&existing).Error; err != nil {
			return err
		}
		oldIDs := make(map[string]struct{}, len(existing))
		newIDs := make(map[string]struct{}, len(refs))
		for _, item := range existing {
			oldIDs[item.TagID] = struct{}{}
		}
		for _, item := range refs {
			newIDs[item.ID] = struct{}{}
		}
		now := time.Now()
		for _, item := range existing {
			if _, keep := newIDs[item.TagID]; keep {
				continue
			}
			oldID := item.TagID
			if err := tx.Unscoped().Where("tenant_id = ? AND document_id = ? AND tag_id = ?", doc.TenantID, doc.ID, item.TagID).Delete(&types.SDPivotDocumentTag{}).Error; err != nil {
				return err
			}
			if err := tx.Create(&types.SDPivotTagAdjustmentLog{ID: uuid.NewString(), TenantID: doc.TenantID, DocumentID: doc.ID, OldTagID: &oldID, Operator: operator, Reason: reason, Status: "applied", CreatedAt: now}).Error; err != nil {
				return err
			}
		}
		for _, ref := range refs {
			if _, keep := oldIDs[ref.ID]; keep {
				continue
			}
			assignment := types.SDPivotDocumentTag{ID: uuid.NewString(), TenantID: doc.TenantID, DocumentID: doc.ID, DimensionID: ref.DimensionID, TagID: ref.ID, Source: "manual", Confidence: 1, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&assignment).Error; err != nil {
				return err
			}
			newID := ref.ID
			if err := tx.Create(&types.SDPivotTagAdjustmentLog{ID: uuid.NewString(), TenantID: doc.TenantID, DocumentID: doc.ID, NewTagID: &newID, Operator: operator, Reason: reason, Status: "applied", CreatedAt: now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
