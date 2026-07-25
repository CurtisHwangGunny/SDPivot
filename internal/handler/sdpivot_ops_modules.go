package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
	if denyIfNotOpsAdmin(c) {
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
	if denyIfNotOpsAdmin(c) {
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
	if denyIfNotOpsAdmin(c) {
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
	if denyIfNotOpsAdmin(c) {
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
	if denyIfNotOpsAdmin(c) {
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
	if denyIfNotOpsAdmin(c) {
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
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	type ModelRow struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		DisplayName string    `json:"display_name"`
		Type        string    `json:"type"`
		Source      string    `json:"source"`
		Description string    `json:"description"`
		IsDefault   bool      `json:"is_default"`
		IsBuiltin   bool      `json:"is_builtin"`
		ManagedBy   string    `json:"managed_by"`
		Status      string    `json:"status"`
		CreatedAt   time.Time `json:"created_at"`
	}

	var models []ModelRow
	h.db.Table("models").
		Select("id, name, display_name, type, source, description, is_default, is_builtin, managed_by, status, created_at").
		Where("deleted_at IS NULL").
		Order("is_default DESC, created_at DESC").
		Find(&models)

	c.JSON(http.StatusOK, gin.H{"models": models})
}

func (h *SDPivotOpsAdminHandler) CreateModel(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		DisplayName string `json:"display_name"`
		Type        string `json:"type" binding:"required"`
		Source      string `json:"source"`
		Description string `json:"description"`
		Parameters  string `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Parameters == "" {
		req.Parameters = "{}"
	}

	model := map[string]interface{}{
		"id":           uuid.New().String(),
		"name":         req.Name,
		"display_name": req.DisplayName,
		"type":         req.Type,
		"source":       req.Source,
		"description":  req.Description,
		"parameters":   req.Parameters,
		"is_default":   false,
		"is_builtin":   false,
		"managed_by":   "ops",
		"tenant_id":    1,
		"status":       "active",
		"created_at":   time.Now(),
		"updated_at":   time.Now(),
	}

	if err := h.db.Table("models").Create(model).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create model"})
		return
	}
	h.writeAuditLog(c, "create_model", "model", model["id"].(string), "name: "+req.Name)
	c.JSON(http.StatusCreated, gin.H{"model": model})
}

func (h *SDPivotOpsAdminHandler) UpdateModel(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")
	var req struct {
		DisplayName *string `json:"display_name"`
		Source      *string `json:"source"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
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
	h.db.Table("models").Where("id = ?", id).Updates(updates)
	h.writeAuditLog(c, "update_model", "model", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "model updated"})
}

func (h *SDPivotOpsAdminHandler) DeleteModel(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")
	h.db.Table("models").Where("id = ? AND is_builtin = false", id).Update("deleted_at", time.Now())
	h.writeAuditLog(c, "delete_model", "model", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "model deleted"})
}

func (h *SDPivotOpsAdminHandler) SetDefaultModel(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")

	// Unset all defaults of same type
	var modelType string
	h.db.Table("models").Where("id = ?", id).Select("type").Row().Scan(&modelType)

	h.db.Table("models").Where("type = ? AND deleted_at IS NULL", modelType).Update("is_default", false)
	h.db.Table("models").Where("id = ?", id).Update("is_default", true)

	h.writeAuditLog(c, "set_default_model", "model", id, "type: "+modelType)
	c.JSON(http.StatusOK, gin.H{"message": "default model set", "model_id": id})
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
