package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tagDictionaryConfig struct {
	Tags []types.TagDictionary `json:"tags"`
}

func normalizeTagDictionaryEntry(entry *types.TagDictionary) error {
	entry.ID = strings.TrimSpace(entry.ID)
	entry.DimensionID = strings.TrimSpace(entry.DimensionID)
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Color = strings.TrimSpace(entry.Color)
	if entry.DimensionID == "" {
		entry.DimensionID = types.DefaultTagDimensionID
	}
	if entry.Name == "" {
		return errors.New("tag name is required")
	}
	if len(entry.Name) > 128 {
		return errors.New("tag name cannot exceed 128 characters")
	}
	if len(entry.Color) > 32 {
		return errors.New("tag color cannot exceed 32 characters")
	}
	return nil
}

func (h *SystemHandler) tagDimensionExists(c *gin.Context, id string) (bool, error) {
	var count int64
	err := h.db.WithContext(c.Request.Context()).Model(&types.TagDimension{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (h *SystemHandler) tagDictionaryNameExists(c *gin.Context, entry types.TagDictionary, excludeID string) (bool, error) {
	var count int64
	db := h.db.WithContext(c.Request.Context()).Model(&types.TagDictionary{}).
		Where("dimension_id = ? AND LOWER(name) = LOWER(?)", entry.DimensionID, entry.Name)
	if excludeID != "" {
		db = db.Where("id <> ?", excludeID)
	}
	err := db.Count(&count).Error
	return count > 0, err
}

func (h *SystemHandler) validateTagDictionaryEntry(c *gin.Context, entry *types.TagDictionary, excludeID string) error {
	if err := normalizeTagDictionaryEntry(entry); err != nil {
		return err
	}
	exists, err := h.tagDimensionExists(c, entry.DimensionID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("tag dimension not found")
	}
	exists, err = h.tagDictionaryNameExists(c, *entry, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("tag name already exists in this dimension")
	}
	return nil
}

func writeTagDictionaryValidationError(c *gin.Context, err error) {
	if strings.Contains(err.Error(), "already exists") {
		c.Error(apperrors.NewConflictError(err.Error()))
		return
	}
	c.Error(apperrors.NewBadRequestError(err.Error()))
}

// ListTagDimensions returns the seven platform tag dimensions.
func (h *SystemHandler) ListTagDimensions(c *gin.Context) {
	var dimensions []types.TagDimension
	if err := h.db.WithContext(c.Request.Context()).Order("sort_order ASC, code ASC").Find(&dimensions).Error; err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to list tag dimensions"))
		return
	}
	c.JSON(http.StatusOK, dimensions)
}

// GetTagDictionary returns the global ordered tag dictionary.
func (h *SystemHandler) GetTagDictionary(c *gin.Context) {
	var tags []types.TagDictionary
	if err := h.db.WithContext(c.Request.Context()).Preload("Dimension").
		Order("dimension_id ASC, sort_order ASC, name ASC").Find(&tags).Error; err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to load tag dictionary"))
		return
	}
	c.JSON(http.StatusOK, tagDictionaryConfig{Tags: tags})
}

// GetTagDictionaryEntry returns one dictionary entry.
func (h *SystemHandler) GetTagDictionaryEntry(c *gin.Context) {
	var entry types.TagDictionary
	if err := h.db.WithContext(c.Request.Context()).Preload("Dimension").First(&entry, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Error(apperrors.NewNotFoundError("Tag dictionary entry not found"))
			return
		}
		c.Error(apperrors.NewInternalServerError("Failed to get tag dictionary entry"))
		return
	}
	c.JSON(http.StatusOK, entry)
}

// CreateTagDictionaryEntry creates one dictionary entry.
func (h *SystemHandler) CreateTagDictionaryEntry(c *gin.Context) {
	var entry types.TagDictionary
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	if err := h.validateTagDictionaryEntry(c, &entry, ""); err != nil {
		writeTagDictionaryValidationError(c, err)
		return
	}
	entry.ID = uuid.NewString()
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = entry.CreatedAt
	if err := h.db.WithContext(c.Request.Context()).Create(&entry).Error; err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to create tag dictionary entry"))
		return
	}
	c.JSON(http.StatusCreated, entry)
}

// UpdateTagDictionaryEntry updates one dictionary entry.
func (h *SystemHandler) UpdateTagDictionaryEntry(c *gin.Context) {
	var current types.TagDictionary
	if err := h.db.WithContext(c.Request.Context()).First(&current, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Error(apperrors.NewNotFoundError("Tag dictionary entry not found"))
			return
		}
		c.Error(apperrors.NewInternalServerError("Failed to get tag dictionary entry"))
		return
	}
	var entry types.TagDictionary
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	entry.ID = current.ID
	if err := h.validateTagDictionaryEntry(c, &entry, current.ID); err != nil {
		writeTagDictionaryValidationError(c, err)
		return
	}
	entry.CreatedAt = current.CreatedAt
	entry.UpdatedAt = time.Now()
	if err := h.db.WithContext(c.Request.Context()).Model(&current).Updates(map[string]any{
		"dimension_id": entry.DimensionID,
		"name":         entry.Name,
		"color":        entry.Color,
		"sort_order":   entry.SortOrder,
		"updated_at":   entry.UpdatedAt,
	}).Error; err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to update tag dictionary entry"))
		return
	}
	c.JSON(http.StatusOK, entry)
}

// DeleteTagDictionaryEntry deletes one dictionary entry and its document assignments.
func (h *SystemHandler) DeleteTagDictionaryEntry(c *gin.Context) {
	result := h.db.WithContext(c.Request.Context()).Delete(&types.TagDictionary{}, "id = ?", c.Param("id"))
	if result.Error != nil {
		c.Error(apperrors.NewInternalServerError("Failed to delete tag dictionary entry"))
		return
	}
	if result.RowsAffected == 0 {
		c.Error(apperrors.NewNotFoundError("Tag dictionary entry not found"))
		return
	}
	c.Status(http.StatusNoContent)
}

// UpdateTagDictionary atomically replaces the global ordered tag dictionary.
func (h *SystemHandler) UpdateTagDictionary(c *gin.Context) {
	var config tagDictionaryConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	if len(config.Tags) > maxTagDictionarySz {
		c.Error(apperrors.NewBadRequestError("tag dictionary cannot contain more than 500 tags"))
		return
	}
	seenIDs := make(map[string]struct{}, len(config.Tags))
	seenNames := make(map[string]struct{}, len(config.Tags))
	for i := range config.Tags {
		if err := normalizeTagDictionaryEntry(&config.Tags[i]); err != nil {
			c.Error(apperrors.NewBadRequestError(err.Error()))
			return
		}
		if config.Tags[i].ID == "" {
			config.Tags[i].ID = uuid.NewString()
		}
		nameKey := config.Tags[i].DimensionID + "\x00" + strings.ToLower(config.Tags[i].Name)
		if _, exists := seenIDs[config.Tags[i].ID]; exists {
			c.Error(apperrors.NewBadRequestError("tag ids must be unique"))
			return
		}
		if _, exists := seenNames[nameKey]; exists {
			c.Error(apperrors.NewBadRequestError("tag names must be unique within each dimension"))
			return
		}
		exists, err := h.tagDimensionExists(c, config.Tags[i].DimensionID)
		if err != nil {
			c.Error(apperrors.NewInternalServerError("Failed to validate tag dimension"))
			return
		}
		if !exists {
			c.Error(apperrors.NewBadRequestError("tag dimension not found"))
			return
		}
		seenIDs[config.Tags[i].ID] = struct{}{}
		seenNames[nameKey] = struct{}{}
	}
	if config.Tags == nil {
		config.Tags = []types.TagDictionary{}
	}
	now := time.Now()
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		ids := make([]string, 0, len(config.Tags))
		for i := range config.Tags {
			config.Tags[i].Dimension = nil
			ids = append(ids, config.Tags[i].ID)
			config.Tags[i].UpdatedAt = now
		}
		deleteQuery := tx.Where("1 = 1")
		if len(ids) > 0 {
			deleteQuery = tx.Where("id NOT IN ?", ids)
		}
		if err := deleteQuery.Delete(&types.TagDictionary{}).Error; err != nil {
			return err
		}
		if len(config.Tags) > 0 {
			return tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"dimension_id", "name", "color", "sort_order", "updated_at",
				}),
			}).Create(&config.Tags).Error
		}
		return nil
	})
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to save tag dictionary"))
		return
	}
	c.JSON(http.StatusOK, config)
}
