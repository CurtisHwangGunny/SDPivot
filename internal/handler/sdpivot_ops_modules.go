package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/handler/dto"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

// ============================================================
// Config — 系统配置（PRD §4.1.3）
// ============================================================

type OpsSystemConfig struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Key         string    `json:"key" gorm:"type:varchar(100);uniqueIndex;not null"`
	Value       string    `json:"value" gorm:"type:text"`
	Description string    `json:"description" gorm:"type:varchar(255)"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (OpsSystemConfig) TableName() string { return "system_configs" }

func (h *SDPivotOpsAdminHandler) ListConfigs(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin") {
		return
	}
	var configs []OpsSystemConfig
	h.db.Order("key ASC").Find(&configs)
	for i := range configs {
		if isSMSSecretConfig(configs[i].Key) && configs[i].Value != "" {
			configs[i].Value = "***"
		}
	}
	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

func (h *SDPivotOpsAdminHandler) UpdateConfig(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin") {
		return
	}
	key := c.Param("key")
	var req struct {
		Value       string `json:"value" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing OpsSystemConfig
	if h.db.Where("key = ?", key).First(&existing).Error == nil {
		updates := map[string]interface{}{"value": req.Value, "updated_at": time.Now()}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		h.db.Model(&existing).Updates(updates)
	} else {
		cfg := OpsSystemConfig{
			Key: key, Value: req.Value, Description: req.Description,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		h.db.Create(&cfg)
	}

	detail := "value: " + req.Value
	if isSMSSecretConfig(key) {
		detail = "secret updated"
	}
	h.writeAuditLog(c, "update_config", "system_config", key, detail)
	responseValue := req.Value
	if isSMSSecretConfig(key) && responseValue != "" {
		responseValue = "***"
	}
	c.JSON(http.StatusOK, gin.H{"message": "config updated", "key": key, "value": responseValue})
}

var smsConfigDefaults = []struct {
	key         string
	value       string
	description string
}{
	{"sms_provider", "custom", "短信服务商: aliyun/tencent/huawei/custom"},
	{"sms_endpoint", "", "短信服务 HTTP 接口地址"},
	{"sms_access_key_id", "", "短信服务 Access Key ID"},
	{"sms_access_key_secret", "", "短信服务 Access Key Secret"},
	{"sms_region", "", "短信服务区域"},
	{"sms_sign_name", "", "短信签名"},
	{"sms_template_id", "", "短信模板 ID"},
	{"sms_app_id", "", "短信应用 ID"},
	{"sms_sender", "", "华为短信发送通道号"},
	{"sms_custom_headers", "{}", "自定义短信接口请求头 JSON"},
	{"sms_timeout_seconds", "10", "短信请求超时秒数"},
}

func isSMSSecretConfig(key string) bool {
	switch key {
	case "sms_access_key_id", "sms_access_key_secret", "sms_custom_headers":
		return true
	default:
		return false
	}
}

func (h *SDPivotOpsAdminHandler) GetSMSConfig(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin") {
		return
	}
	response := make(map[string]interface{}, len(smsConfigDefaults)+3)
	for _, item := range smsConfigDefaults {
		value := h.getOrCreateConfigValue(item.key, item.value, item.description)
		if isSMSSecretConfig(item.key) {
			response[item.key+"_configured"] = value != "" && value != "{}"
			continue
		}
		response[item.key] = value
	}
	c.JSON(http.StatusOK, response)
}

func (h *SDPivotOpsAdminHandler) UpdateSMSConfig(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin") {
		return
	}
	var req struct {
		Provider        string  `json:"provider" binding:"required,oneof=aliyun tencent huawei custom"`
		Endpoint        string  `json:"endpoint" binding:"required,url"`
		AccessKeyID     *string `json:"access_key_id"`
		AccessKeySecret *string `json:"access_key_secret"`
		Region          string  `json:"region"`
		SignName        string  `json:"sign_name"`
		TemplateID      string  `json:"template_id"`
		AppID           string  `json:"app_id"`
		Sender          string  `json:"sender"`
		CustomHeaders   *string `json:"custom_headers"`
		TimeoutSeconds  int     `json:"timeout_seconds" binding:"omitempty,min=1,max=120"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 10
	}

	values := map[string]string{
		"sms_provider":        strings.ToLower(strings.TrimSpace(req.Provider)),
		"sms_endpoint":        strings.TrimSpace(req.Endpoint),
		"sms_region":          strings.TrimSpace(req.Region),
		"sms_sign_name":       strings.TrimSpace(req.SignName),
		"sms_template_id":     strings.TrimSpace(req.TemplateID),
		"sms_app_id":          strings.TrimSpace(req.AppID),
		"sms_sender":          strings.TrimSpace(req.Sender),
		"sms_timeout_seconds": strconv.Itoa(req.TimeoutSeconds),
	}
	if req.AccessKeyID != nil {
		values["sms_access_key_id"] = strings.TrimSpace(*req.AccessKeyID)
	}
	if req.AccessKeySecret != nil {
		values["sms_access_key_secret"] = strings.TrimSpace(*req.AccessKeySecret)
	}
	if req.CustomHeaders != nil {
		values["sms_custom_headers"] = strings.TrimSpace(*req.CustomHeaders)
	}
	for _, item := range smsConfigDefaults {
		if value, ok := values[item.key]; ok {
			h.upsertConfig(item.key, value, item.description)
		}
	}
	h.writeAuditLog(c, "update_sms_config", "system_config", "sms", "provider: "+values["sms_provider"])
	c.JSON(http.StatusOK, gin.H{"message": "sms config updated"})
}

func (h *SDPivotOpsAdminHandler) GetTrialConfig(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin") {
		return
	}
	trialDays := h.getOrCreateConfigValue("trial_days", "30", "试用期天数")
	extendedDays := h.getOrCreateConfigValue("extended_trial_days", "90", "认证后延长天数")
	c.JSON(http.StatusOK, gin.H{
		"trial_days":          trialDays,
		"extended_trial_days": extendedDays,
	})
}

func (h *SDPivotOpsAdminHandler) UpdateTrialConfig(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin") {
		return
	}
	var req struct {
		TrialDays         int `json:"trial_days"`
		ExtendedTrialDays int `json:"extended_trial_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.upsertConfig("trial_days", strconv.Itoa(req.TrialDays), "试用期天数")
	h.upsertConfig("extended_trial_days", strconv.Itoa(req.ExtendedTrialDays), "认证后延长天数")

	h.writeAuditLog(c, "update_trial_config", "system_config", "trial", "")
	c.JSON(http.StatusOK, gin.H{"message": "trial config updated"})
}

func (h *SDPivotOpsAdminHandler) upsertConfig(key, value, description string) {
	var existing OpsSystemConfig
	if err := h.db.Where("key = ?", key).First(&existing).Error; err == nil {
		h.db.Model(&existing).Updates(map[string]interface{}{"value": value, "updated_at": time.Now()})
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		h.db.Create(&OpsSystemConfig{Key: key, Value: value, Description: description, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	}
}

func (h *SDPivotOpsAdminHandler) getOrCreateConfigValue(key, defaultValue, description string) string {
	var cfg OpsSystemConfig
	if err := h.db.Where("key = ?", key).First(&cfg).Error; err == nil {
		if cfg.Value != "" {
			return cfg.Value
		}
		cfg.Value = defaultValue
		if cfg.Description == "" {
			cfg.Description = description
		}
		h.db.Model(&cfg).Updates(map[string]interface{}{
			"value":       cfg.Value,
			"description": cfg.Description,
			"updated_at":  time.Now(),
		})
		return cfg.Value
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = OpsSystemConfig{Key: key, Value: defaultValue, Description: description, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		h.db.Create(&cfg)
		return cfg.Value
	}
	return defaultValue
}

// ============================================================
// Models — 模型管理（PRD §4.1.3）
// ============================================================

func (h *SDPivotOpsAdminHandler) ListModels(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin", "department_admin") {
		return
	}

	models := make([]*types.Model, 0)
	query := h.db.Where("deleted_at IS NULL")
	if strings.Contains(c.FullPath(), "/admin/models") {
		query = query.Where("tenant_id = ? OR is_builtin = true", middleware.GetTenantID(c))
	} else if query.Dialector.Name() == "postgres" {
		query.Exec("SET LOCAL row_security = off")
	}
	if err := query.
		Order("is_default DESC, created_at DESC").
		Find(&models).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": dto.NewModelResponses(models)})
}

func (h *SDPivotOpsAdminHandler) CreateModel(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin", "department_admin") {
		return
	}
	var req struct {
		Name        string            `json:"name" binding:"required"`
		DisplayName string            `json:"display_name"`
		Type        types.ModelType   `json:"type" binding:"required"`
		Source      types.ModelSource `json:"source" binding:"required"`
		Description string            `json:"description"`
		Parameters  json.RawMessage   `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	parameters, err := parseModelParameters(req.Parameters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parameters must be a JSON object"})
		return
	}
	if parameters.BaseURL != "" {
		if err := secutils.ValidateURLForSSRF(parameters.BaseURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or unsafe model endpoint"})
			return
		}
	}
	now := time.Now()
	model := types.Model{
		ID: uuid.NewString(), TenantID: middleware.GetTenantID(c), Name: strings.TrimSpace(req.Name),
		DisplayName: strings.TrimSpace(req.DisplayName), Type: req.Type, Source: req.Source,
		Description: strings.TrimSpace(req.Description), Parameters: parameters, IsDefault: false,
		IsBuiltin: false, ManagedBy: "ops", Status: types.ModelStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if model.TenantID == 0 {
		model.TenantID = types.DefaultBuiltinModelTenantID
	}
	if err := h.db.Create(&model).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create model"})
		return
	}
	h.writeAuditLog(c, "create_model", "model", model.ID, "name: "+req.Name)
	c.JSON(http.StatusCreated, gin.H{"model": dto.NewModelResponse(&model)})
}

func (h *SDPivotOpsAdminHandler) UpdateModel(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin", "department_admin") {
		return
	}
	id := c.Param("id")
	var req struct {
		Name        *string            `json:"name"`
		DisplayName *string            `json:"display_name"`
		Type        *types.ModelType   `json:"type"`
		Source      *types.ModelSource `json:"source"`
		Description *string            `json:"description"`
		Status      *types.ModelStatus `json:"status"`
		Parameters  json.RawMessage    `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var model types.Model
	if err := h.db.Where("id = ? AND tenant_id = ? AND is_builtin = false AND deleted_at IS NULL", id, middleware.GetTenantID(c)).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load model"})
		}
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.DisplayName != nil {
		updates["display_name"] = strings.TrimSpace(*req.DisplayName)
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Source != nil {
		updates["source"] = *req.Source
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(req.Parameters) > 0 && string(req.Parameters) != "null" {
		parameters, err := parseModelParameters(req.Parameters)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parameters must be a JSON object"})
			return
		}
		if parameters.APIKey == "" {
			parameters.APIKey = model.Parameters.APIKey
		}
		if parameters.AppSecret == "" {
			parameters.AppSecret = model.Parameters.AppSecret
		}
		if parameters.BaseURL != "" {
			if err := secutils.ValidateURLForSSRF(parameters.BaseURL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or unsafe model endpoint"})
				return
			}
		}
		updates["parameters"] = parameters
	}
	result := h.db.Model(&types.Model{}).Where("id = ? AND tenant_id = ? AND is_builtin = false", id, middleware.GetTenantID(c)).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update model"})
		return
	}
	h.writeAuditLog(c, "update_model", "model", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "model updated"})
}

func (h *SDPivotOpsAdminHandler) DeleteModel(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin") {
		return
	}
	id := c.Param("id")
	h.db.Table("models").Where("id = ? AND is_builtin = false", id).Update("deleted_at", time.Now())
	h.writeAuditLog(c, "delete_model", "model", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "model deleted"})
}

func (h *SDPivotOpsAdminHandler) SetDefaultModel(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin", "department_admin") {
		return
	}
	id := c.Param("id")
	tenantID := middleware.GetTenantID(c)
	var model types.Model
	if err := h.db.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL AND status = ?", id, tenantID, types.ModelStatusActive).First(&model).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "active model not found"})
		return
	}
	multiModel := h.multiModelEnabled(tenantID)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if !multiModel {
			if err := tx.Model(&types.Model{}).Where("tenant_id = ? AND type = ? AND deleted_at IS NULL", tenantID, model.Type).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Model(&types.Model{}).Where("id = ? AND tenant_id = ?", id, tenantID).Update("is_default", true).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set default model"})
		return
	}
	h.writeAuditLog(c, "set_default_model", "model", id, "type: "+string(model.Type))
	c.JSON(http.StatusOK, gin.H{"message": "default model set", "model_id": id, "multi_model_enabled": multiModel})
}

func (h *SDPivotOpsAdminHandler) TestModel(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin", "department_admin") {
		return
	}
	id := c.Param("id")
	tenantID := middleware.GetTenantID(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	result, err := NewSDPivotLLMService(h.db).GenerateWithModel(ctx, tenantID, id, "You are a connectivity check.", "Reply with OK.", 8)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "model connectivity test failed", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "model connection succeeded", "model_id": id, "response": result.Content})
}

func parseModelParameters(raw json.RawMessage) (types.ModelParameters, error) {
	var parameters types.ModelParameters
	if len(raw) == 0 || string(raw) == "null" {
		return parameters, nil
	}
	if raw[0] == '"' {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return parameters, err
		}
		raw = json.RawMessage(encoded)
	}
	if err := json.Unmarshal(raw, &parameters); err != nil {
		return parameters, err
	}
	return parameters, nil
}

func (h *SDPivotOpsAdminHandler) multiModelEnabled(tenantID uint64) bool {
	var value string
	err := h.db.Table("system_settings").Select("value").
		Where("tenant_id = ? AND section = ? AND key = ?", tenantID, "global", "multi_model_enabled").
		Scan(&value).Error
	if err != nil {
		return false
	}
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

// ============================================================
// Helper
// ============================================================

func getUserIDFromCtx(c *gin.Context) string {
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
