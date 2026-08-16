package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const modelConfigKeyPrefix = "model_config:"

type modelConfigRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	Key         string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Value       string    `gorm:"type:text;not null;default:''"`
	Description string    `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (modelConfigRow) TableName() string { return "system_configs" }

type modelConfigRequest struct {
	Name        string   `json:"name" binding:"required"`
	Provider    string   `json:"provider" binding:"required"`
	Endpoint    string   `json:"endpoint"`
	APIKey      *string  `json:"api_key"`
	Temperature *float64 `json:"temperature"`
	MaxTokens   *int     `json:"max_tokens"`
	TopP        *float64 `json:"top_p"`
}

type storedModelConfig struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Provider    string    `json:"provider"`
	Endpoint    string    `json:"endpoint"`
	APIKey      string    `json:"api_key"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	TopP        float64   `json:"top_p"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type modelConfigResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Provider         string    `json:"provider"`
	Endpoint         string    `json:"endpoint"`
	APIKeyConfigured bool      `json:"api_key_configured"`
	Temperature      float64   `json:"temperature"`
	MaxTokens        int       `json:"max_tokens"`
	TopP             float64   `json:"top_p"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func newModelConfigResponse(config storedModelConfig) modelConfigResponse {
	return modelConfigResponse{
		ID: config.ID, Name: config.Name, Provider: config.Provider, Endpoint: config.Endpoint,
		APIKeyConfigured: config.APIKey != "", Temperature: config.Temperature, MaxTokens: config.MaxTokens,
		TopP: config.TopP, CreatedAt: config.CreatedAt, UpdatedAt: config.UpdatedAt,
	}
}

func normalizeModelConfigRequest(req *modelConfigRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Provider = strings.TrimSpace(req.Provider)
	req.Endpoint = strings.TrimSpace(req.Endpoint)
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.Provider == "" {
		return errors.New("provider is required")
	}
	if req.Endpoint != "" {
		if err := secutils.ValidateURLForSSRF(req.Endpoint); err != nil {
			return errors.New(secutils.FormatSSRFError("endpoint", req.Endpoint, err))
		}
	}
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return errors.New("temperature must be between 0 and 2")
	}
	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		return errors.New("max_tokens must be greater than 0")
	}
	if req.TopP != nil && (*req.TopP < 0 || *req.TopP > 1) {
		return errors.New("top_p must be between 0 and 1")
	}
	if req.APIKey != nil {
		trimmed := strings.TrimSpace(*req.APIKey)
		req.APIKey = &trimmed
	}
	return nil
}

func decodeModelConfig(row *modelConfigRow) (storedModelConfig, error) {
	var config storedModelConfig
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		return storedModelConfig{}, err
	}
	if config.ID == "" {
		config.ID = strings.TrimPrefix(row.Key, modelConfigKeyPrefix)
	}
	return config, nil
}

func encodeModelConfig(config storedModelConfig) (string, error) {
	value, err := json.Marshal(config)
	return string(value), err
}

func (h *SystemHandler) findModelConfigRow(c *gin.Context) (*modelConfigRow, storedModelConfig, error) {
	var row modelConfigRow
	err := h.db.WithContext(c.Request.Context()).Where("key = ?", modelConfigKeyPrefix+c.Param("id")).First(&row).Error
	if err != nil {
		return nil, storedModelConfig{}, err
	}
	config, err := decodeModelConfig(&row)
	return &row, config, err
}

func (h *SystemHandler) modelConfigNameExists(name, excludeID string) (bool, error) {
	var rows []modelConfigRow
	if err := h.db.Where("key LIKE ?", modelConfigKeyPrefix+"%").Find(&rows).Error; err != nil {
		return false, err
	}
	for i := range rows {
		config, err := decodeModelConfig(&rows[i])
		if err != nil {
			return false, err
		}
		if config.ID != excludeID && strings.EqualFold(config.Name, name) {
			return true, nil
		}
	}
	return false, nil
}

// ListModelConfigs returns all platform-level model provider configurations.
func (h *SystemHandler) ListModelConfigs(c *gin.Context) {
	var rows []modelConfigRow
	if err := h.db.WithContext(c.Request.Context()).Where("key LIKE ?", modelConfigKeyPrefix+"%").Order("key ASC").Find(&rows).Error; err != nil {
		logger.Errorf(c.Request.Context(), "list model configs failed: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list model configurations"))
		return
	}
	responses := make([]modelConfigResponse, 0, len(rows))
	for i := range rows {
		config, err := decodeModelConfig(&rows[i])
		if err != nil {
			logger.Errorf(c.Request.Context(), "decode model config %q failed: %v", rows[i].Key, err)
			c.Error(apperrors.NewInternalServerError("Failed to decode model configuration"))
			return
		}
		responses = append(responses, newModelConfigResponse(config))
	}
	c.JSON(http.StatusOK, responses)
}

// GetModelConfig returns one platform-level model provider configuration.
func (h *SystemHandler) GetModelConfig(c *gin.Context) {
	_, config, err := h.findModelConfigRow(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Error(apperrors.NewNotFoundError("Model configuration not found"))
			return
		}
		logger.Errorf(c.Request.Context(), "get model config failed: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to get model configuration"))
		return
	}
	c.JSON(http.StatusOK, newModelConfigResponse(config))
}

// CreateModelConfig creates a platform-level model provider configuration.
func (h *SystemHandler) CreateModelConfig(c *gin.Context) {
	var req modelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	if err := normalizeModelConfigRequest(&req); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	exists, err := h.modelConfigNameExists(req.Name, "")
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to validate model configuration"))
		return
	}
	if exists {
		c.Error(apperrors.NewConflictError("Model configuration name already exists"))
		return
	}
	now := time.Now()
	config := storedModelConfig{ID: uuid.NewString(), Name: req.Name, Provider: req.Provider, Endpoint: req.Endpoint, Temperature: 0.7, MaxTokens: 2048, TopP: 1, CreatedAt: now, UpdatedAt: now}
	if req.Temperature != nil {
		config.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		config.MaxTokens = *req.MaxTokens
	}
	if req.TopP != nil {
		config.TopP = *req.TopP
	}
	if req.APIKey != nil {
		config.APIKey, err = secutils.EncryptAESGCM(*req.APIKey, secutils.GetAESKey())
		if err != nil {
			c.Error(apperrors.NewInternalServerError("Failed to encrypt API key"))
			return
		}
	}
	value, err := encodeModelConfig(config)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to encode model configuration"))
		return
	}
	row := modelConfigRow{Key: modelConfigKeyPrefix + config.ID, Value: value, Description: "AI model configuration: " + config.Name, CreatedAt: now, UpdatedAt: now}
	if err := h.db.WithContext(c.Request.Context()).Create(&row).Error; err != nil {
		logger.Errorf(c.Request.Context(), "create model config failed: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to create model configuration"))
		return
	}
	c.JSON(http.StatusCreated, newModelConfigResponse(config))
}

// UpdateModelConfig replaces a platform-level model provider configuration.
func (h *SystemHandler) UpdateModelConfig(c *gin.Context) {
	row, config, err := h.findModelConfigRow(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Error(apperrors.NewNotFoundError("Model configuration not found"))
			return
		}
		c.Error(apperrors.NewInternalServerError("Failed to get model configuration"))
		return
	}
	var req modelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	if err := normalizeModelConfigRequest(&req); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	exists, err := h.modelConfigNameExists(req.Name, config.ID)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to validate model configuration"))
		return
	}
	if exists {
		c.Error(apperrors.NewConflictError("Model configuration name already exists"))
		return
	}
	config.Name, config.Provider, config.Endpoint = req.Name, req.Provider, req.Endpoint
	if req.Temperature != nil {
		config.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		config.MaxTokens = *req.MaxTokens
	}
	if req.TopP != nil {
		config.TopP = *req.TopP
	}
	if req.APIKey != nil {
		config.APIKey, err = secutils.EncryptAESGCM(*req.APIKey, secutils.GetAESKey())
		if err != nil {
			c.Error(apperrors.NewInternalServerError("Failed to encrypt API key"))
			return
		}
	}
	config.UpdatedAt = time.Now()
	value, err := encodeModelConfig(config)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to encode model configuration"))
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Model(row).Updates(map[string]any{"value": value, "description": "AI model configuration: " + config.Name, "updated_at": config.UpdatedAt}).Error; err != nil {
		logger.Errorf(c.Request.Context(), "update model config failed: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to update model configuration"))
		return
	}
	c.JSON(http.StatusOK, newModelConfigResponse(config))
}

// DeleteModelConfig deletes a platform-level model provider configuration.
func (h *SystemHandler) DeleteModelConfig(c *gin.Context) {
	result := h.db.WithContext(c.Request.Context()).Where("key = ?", modelConfigKeyPrefix+c.Param("id")).Delete(&modelConfigRow{})
	if result.Error != nil {
		c.Error(apperrors.NewInternalServerError("Failed to delete model configuration"))
		return
	}
	if result.RowsAffected == 0 {
		c.Error(apperrors.NewNotFoundError("Model configuration not found"))
		return
	}
	c.Status(http.StatusNoContent)
}
