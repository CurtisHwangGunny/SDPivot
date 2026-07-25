package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	storageConfigKey   = "storage_config"
	smsConfigKey       = "sms_config"
	wechatLoginKey     = "wechat_login_config"
	tagDictionaryKey   = "tag_dictionary"
	globalParamsKey    = "global_params"
	defaultConfigDesc  = "Platform system configuration"
	maxTagDictionarySz = 500
)

type systemConfigRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	Key         string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Value       string    `gorm:"type:text;not null;default:''"`
	Description string    `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (systemConfigRow) TableName() string { return "system_configs" }

type storageConfig struct {
	Provider        string `json:"provider"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	UseSSL          bool   `json:"use_ssl"`
	ForcePathStyle  bool   `json:"force_path_style"`
	PathPrefix      string `json:"path_prefix"`
}

type storageConfigRequest struct {
	Provider        string  `json:"provider" binding:"required"`
	Endpoint        string  `json:"endpoint" binding:"required"`
	Region          string  `json:"region"`
	Bucket          string  `json:"bucket" binding:"required"`
	AccessKeyID     *string `json:"access_key_id"`
	SecretAccessKey *string `json:"secret_access_key"`
	UseSSL          bool    `json:"use_ssl"`
	ForcePathStyle  bool    `json:"force_path_style"`
	PathPrefix      string  `json:"path_prefix"`
}

type storageConfigResponse struct {
	Provider                  string `json:"provider"`
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKeyIDConfigured     bool   `json:"access_key_id_configured"`
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"`
	UseSSL                    bool   `json:"use_ssl"`
	ForcePathStyle            bool   `json:"force_path_style"`
	PathPrefix                string `json:"path_prefix"`
}

type smsSystemConfig struct {
	Provider        string            `json:"provider"`
	Endpoint        string            `json:"endpoint"`
	AccessKeyID     string            `json:"access_key_id,omitempty"`
	AccessKeySecret string            `json:"access_key_secret,omitempty"`
	Region          string            `json:"region"`
	SignName        string            `json:"sign_name"`
	TemplateID      string            `json:"template_id"`
	AppID           string            `json:"app_id"`
	Sender          string            `json:"sender"`
	CustomHeaders   map[string]string `json:"custom_headers"`
	TimeoutSeconds  int               `json:"timeout_seconds"`
}

type smsConfigRequest struct {
	Provider        string             `json:"provider" binding:"required"`
	Endpoint        string             `json:"endpoint" binding:"required"`
	AccessKeyID     *string            `json:"access_key_id"`
	AccessKeySecret *string            `json:"access_key_secret"`
	Region          string             `json:"region"`
	SignName        string             `json:"sign_name"`
	TemplateID      string             `json:"template_id"`
	AppID           string             `json:"app_id"`
	Sender          string             `json:"sender"`
	CustomHeaders   *map[string]string `json:"custom_headers"`
	TimeoutSeconds  int                `json:"timeout_seconds"`
}

type smsConfigResponse struct {
	Provider                  string            `json:"provider"`
	Endpoint                  string            `json:"endpoint"`
	AccessKeyIDConfigured     bool              `json:"access_key_id_configured"`
	AccessKeySecretConfigured bool              `json:"access_key_secret_configured"`
	Region                    string            `json:"region"`
	SignName                  string            `json:"sign_name"`
	TemplateID                string            `json:"template_id"`
	AppID                     string            `json:"app_id"`
	Sender                    string            `json:"sender"`
	CustomHeaders             map[string]string `json:"custom_headers"`
	TimeoutSeconds            int               `json:"timeout_seconds"`
}

type wechatLoginConfig struct {
	Enabled     bool   `json:"enabled"`
	AppID       string `json:"app_id"`
	AppSecret   string `json:"app_secret,omitempty"`
	RedirectURL string `json:"redirect_url"`
}

type wechatLoginConfigRequest struct {
	Enabled     bool    `json:"enabled"`
	AppID       string  `json:"app_id"`
	AppSecret   *string `json:"app_secret"`
	RedirectURL string  `json:"redirect_url"`
}

type wechatLoginConfigResponse struct {
	Enabled             bool   `json:"enabled"`
	AppID               string `json:"app_id"`
	AppSecretConfigured bool   `json:"app_secret_configured"`
	RedirectURL         string `json:"redirect_url"`
}

type tagDictionaryEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

type tagDictionaryConfig struct {
	Tags []tagDictionaryEntry `json:"tags"`
}

type globalParamsConfig struct {
	ChunkSize   int     `json:"chunk_size"`
	Threshold   float64 `json:"threshold"`
	TokenLimit  int     `json:"token_limit"`
	Concurrency int     `json:"concurrency"`
}

func defaultStorageConfig() storageConfig {
	return storageConfig{Provider: "minio", UseSSL: false, ForcePathStyle: true}
}

func defaultSMSSystemConfig() smsSystemConfig {
	return smsSystemConfig{Provider: "custom", CustomHeaders: map[string]string{}, TimeoutSeconds: 10}
}

func defaultGlobalParamsConfig() globalParamsConfig {
	return globalParamsConfig{ChunkSize: 512, Threshold: 0.5, TokenLimit: 4096, Concurrency: 32}
}

func (h *SystemHandler) loadSystemConfig(c *gin.Context, key string, target any) error {
	var row systemConfigRow
	err := h.db.WithContext(c.Request.Context()).Where("key = ?", key).First(&row).Error
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(row.Value), target); err != nil {
		return err
	}
	return nil
}

func (h *SystemHandler) saveSystemConfig(c *gin.Context, key, description string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	now := time.Now()
	row := systemConfigRow{Key: key, Value: string(encoded), Description: description, CreatedAt: now, UpdatedAt: now}
	return h.db.WithContext(c.Request.Context()).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"value":       row.Value,
			"description": row.Description,
			"updated_at":  now,
		}),
	}).Create(&row).Error
}

func loadConfigOrDefault[T any](h *SystemHandler, c *gin.Context, key string, fallback T) (T, error) {
	value := fallback
	err := h.loadSystemConfig(c, key, &value)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fallback, nil
	}
	return value, err
}

func validateHTTPURL(raw, field string) error {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New(field + " must be a valid HTTP or HTTPS URL")
	}
	return nil
}

func encryptOptionalSecret(value *string, current string) (string, error) {
	if value == nil {
		return current, nil
	}
	return secutils.EncryptAESGCM(strings.TrimSpace(*value), secutils.GetAESKey())
}

func storageResponse(config storageConfig) storageConfigResponse {
	return storageConfigResponse{
		Provider: config.Provider, Endpoint: config.Endpoint, Region: config.Region, Bucket: config.Bucket,
		AccessKeyIDConfigured: config.AccessKeyID != "", SecretAccessKeyConfigured: config.SecretAccessKey != "",
		UseSSL: config.UseSSL, ForcePathStyle: config.ForcePathStyle, PathPrefix: config.PathPrefix,
	}
}

func smsResponse(config smsSystemConfig) smsConfigResponse {
	headers := config.CustomHeaders
	if headers == nil {
		headers = map[string]string{}
	}
	return smsConfigResponse{
		Provider: config.Provider, Endpoint: config.Endpoint,
		AccessKeyIDConfigured: config.AccessKeyID != "", AccessKeySecretConfigured: config.AccessKeySecret != "",
		Region: config.Region, SignName: config.SignName, TemplateID: config.TemplateID, AppID: config.AppID,
		Sender: config.Sender, CustomHeaders: headers, TimeoutSeconds: config.TimeoutSeconds,
	}
}

func wechatLoginResponse(config wechatLoginConfig) wechatLoginConfigResponse {
	return wechatLoginConfigResponse{
		Enabled: config.Enabled, AppID: config.AppID, AppSecretConfigured: config.AppSecret != "", RedirectURL: config.RedirectURL,
	}
}

// GetStorageConfig returns the platform S3/MinIO configuration without secrets.
func (h *SystemHandler) GetStorageConfig(c *gin.Context) {
	config, err := loadConfigOrDefault(h, c, storageConfigKey, defaultStorageConfig())
	if err != nil {
		logger.Errorf(c.Request.Context(), "load storage config failed: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to load storage configuration"))
		return
	}
	c.JSON(http.StatusOK, storageResponse(config))
}

// UpdateStorageConfig replaces the platform S3/MinIO configuration.
func (h *SystemHandler) UpdateStorageConfig(c *gin.Context) {
	var req storageConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.Endpoint = strings.TrimSpace(req.Endpoint)
	req.Region = strings.TrimSpace(req.Region)
	req.Bucket = strings.TrimSpace(req.Bucket)
	req.PathPrefix = strings.Trim(strings.TrimSpace(req.PathPrefix), "/")
	if req.Provider != "s3" && req.Provider != "minio" {
		c.Error(apperrors.NewBadRequestError("provider must be s3 or minio"))
		return
	}
	if err := validateHTTPURL(req.Endpoint, "endpoint"); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	current, err := loadConfigOrDefault(h, c, storageConfigKey, defaultStorageConfig())
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to load storage configuration"))
		return
	}
	accessKey, err := encryptOptionalSecret(req.AccessKeyID, current.AccessKeyID)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to encrypt storage access key"))
		return
	}
	secretKey, err := encryptOptionalSecret(req.SecretAccessKey, current.SecretAccessKey)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to encrypt storage secret key"))
		return
	}
	config := storageConfig{
		Provider: req.Provider, Endpoint: req.Endpoint, Region: req.Region, Bucket: req.Bucket,
		AccessKeyID: accessKey, SecretAccessKey: secretKey, UseSSL: req.UseSSL,
		ForcePathStyle: req.ForcePathStyle, PathPrefix: req.PathPrefix,
	}
	if err := h.saveSystemConfig(c, storageConfigKey, "Platform S3/MinIO storage configuration", config); err != nil {
		logger.Errorf(c.Request.Context(), "save storage config failed: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to save storage configuration"))
		return
	}
	c.JSON(http.StatusOK, storageResponse(config))
}

// GetSMSSystemConfig returns the platform SMS provider configuration without secrets.
func (h *SystemHandler) GetSMSSystemConfig(c *gin.Context) {
	config, err := loadConfigOrDefault(h, c, smsConfigKey, defaultSMSSystemConfig())
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to load SMS configuration"))
		return
	}
	c.JSON(http.StatusOK, smsResponse(config))
}

// UpdateSMSSystemConfig replaces the platform SMS provider configuration.
func (h *SystemHandler) UpdateSMSSystemConfig(c *gin.Context) {
	var req smsConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.Endpoint = strings.TrimSpace(req.Endpoint)
	if req.Provider != "aliyun" && req.Provider != "tencent" && req.Provider != "huawei" && req.Provider != "custom" {
		c.Error(apperrors.NewBadRequestError("provider must be aliyun, tencent, huawei, or custom"))
		return
	}
	if err := validateHTTPURL(req.Endpoint, "endpoint"); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 10
	}
	if req.TimeoutSeconds < 1 || req.TimeoutSeconds > 120 {
		c.Error(apperrors.NewBadRequestError("timeout_seconds must be between 1 and 120"))
		return
	}
	current, err := loadConfigOrDefault(h, c, smsConfigKey, defaultSMSSystemConfig())
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to load SMS configuration"))
		return
	}
	accessKey, err := encryptOptionalSecret(req.AccessKeyID, current.AccessKeyID)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to encrypt SMS access key"))
		return
	}
	secret, err := encryptOptionalSecret(req.AccessKeySecret, current.AccessKeySecret)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to encrypt SMS secret"))
		return
	}
	headers := current.CustomHeaders
	if req.CustomHeaders != nil {
		headers = *req.CustomHeaders
	}
	config := smsSystemConfig{
		Provider: req.Provider, Endpoint: req.Endpoint, AccessKeyID: accessKey, AccessKeySecret: secret,
		Region: strings.TrimSpace(req.Region), SignName: strings.TrimSpace(req.SignName),
		TemplateID: strings.TrimSpace(req.TemplateID), AppID: strings.TrimSpace(req.AppID),
		Sender: strings.TrimSpace(req.Sender), CustomHeaders: headers, TimeoutSeconds: req.TimeoutSeconds,
	}
	if err := h.saveSystemConfig(c, smsConfigKey, "Platform SMS provider configuration", config); err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to save SMS configuration"))
		return
	}
	c.JSON(http.StatusOK, smsResponse(config))
}

// GetWeChatLoginConfig returns the platform WeChat login configuration without its secret.
func (h *SystemHandler) GetWeChatLoginConfig(c *gin.Context) {
	config, err := loadConfigOrDefault(h, c, wechatLoginKey, wechatLoginConfig{})
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to load WeChat login configuration"))
		return
	}
	c.JSON(http.StatusOK, wechatLoginResponse(config))
}

// UpdateWeChatLoginConfig replaces the platform WeChat login configuration.
func (h *SystemHandler) UpdateWeChatLoginConfig(c *gin.Context) {
	var req wechatLoginConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	req.AppID = strings.TrimSpace(req.AppID)
	req.RedirectURL = strings.TrimSpace(req.RedirectURL)
	if req.Enabled && (req.AppID == "" || req.RedirectURL == "") {
		c.Error(apperrors.NewBadRequestError("app_id and redirect_url are required when WeChat login is enabled"))
		return
	}
	if req.RedirectURL != "" {
		if err := validateHTTPURL(req.RedirectURL, "redirect_url"); err != nil {
			c.Error(apperrors.NewBadRequestError(err.Error()))
			return
		}
	}
	current, err := loadConfigOrDefault(h, c, wechatLoginKey, wechatLoginConfig{})
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to load WeChat login configuration"))
		return
	}
	secret, err := encryptOptionalSecret(req.AppSecret, current.AppSecret)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to encrypt WeChat app secret"))
		return
	}
	config := wechatLoginConfig{Enabled: req.Enabled, AppID: req.AppID, AppSecret: secret, RedirectURL: req.RedirectURL}
	if err := h.saveSystemConfig(c, wechatLoginKey, "Platform WeChat login configuration", config); err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to save WeChat login configuration"))
		return
	}
	c.JSON(http.StatusOK, wechatLoginResponse(config))
}

// GetTagDictionary returns the global ordered tag dictionary.
func (h *SystemHandler) GetTagDictionary(c *gin.Context) {
	config, err := loadConfigOrDefault(h, c, tagDictionaryKey, tagDictionaryConfig{Tags: []tagDictionaryEntry{}})
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to load tag dictionary"))
		return
	}
	if config.Tags == nil {
		config.Tags = []tagDictionaryEntry{}
	}
	c.JSON(http.StatusOK, config)
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
		config.Tags[i].ID = strings.TrimSpace(config.Tags[i].ID)
		config.Tags[i].Name = strings.TrimSpace(config.Tags[i].Name)
		config.Tags[i].Color = strings.TrimSpace(config.Tags[i].Color)
		if config.Tags[i].Name == "" {
			c.Error(apperrors.NewBadRequestError("tag name is required"))
			return
		}
		if config.Tags[i].ID == "" {
			config.Tags[i].ID = uuid.NewString()
		}
		nameKey := strings.ToLower(config.Tags[i].Name)
		if _, exists := seenIDs[config.Tags[i].ID]; exists {
			c.Error(apperrors.NewBadRequestError("tag ids must be unique"))
			return
		}
		if _, exists := seenNames[nameKey]; exists {
			c.Error(apperrors.NewBadRequestError("tag names must be unique"))
			return
		}
		seenIDs[config.Tags[i].ID] = struct{}{}
		seenNames[nameKey] = struct{}{}
	}
	sort.SliceStable(config.Tags, func(i, j int) bool { return config.Tags[i].SortOrder < config.Tags[j].SortOrder })
	if config.Tags == nil {
		config.Tags = []tagDictionaryEntry{}
	}
	if err := h.saveSystemConfig(c, tagDictionaryKey, "Global tag dictionary", config); err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to save tag dictionary"))
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetGlobalParams returns global processing defaults.
func (h *SystemHandler) GetGlobalParams(c *gin.Context) {
	config, err := loadConfigOrDefault(h, c, globalParamsKey, defaultGlobalParamsConfig())
	if err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to load global parameters"))
		return
	}
	c.JSON(http.StatusOK, config)
}

// UpdateGlobalParams replaces global processing defaults.
func (h *SystemHandler) UpdateGlobalParams(c *gin.Context) {
	var config globalParamsConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.Error(apperrors.NewBadRequestError(err.Error()))
		return
	}
	if config.ChunkSize < 1 || config.ChunkSize > 65536 {
		c.Error(apperrors.NewBadRequestError("chunk_size must be between 1 and 65536"))
		return
	}
	if config.Threshold < 0 || config.Threshold > 1 {
		c.Error(apperrors.NewBadRequestError("threshold must be between 0 and 1"))
		return
	}
	if config.TokenLimit < 0 || config.TokenLimit > 1000000 {
		c.Error(apperrors.NewBadRequestError("token_limit must be between 0 and 1000000"))
		return
	}
	if config.Concurrency < 1 || config.Concurrency > 1024 {
		c.Error(apperrors.NewBadRequestError("concurrency must be between 1 and 1024"))
		return
	}
	if err := h.saveSystemConfig(c, globalParamsKey, defaultConfigDesc+": global processing parameters", config); err != nil {
		c.Error(apperrors.NewInternalServerError("Failed to save global parameters"))
		return
	}
	c.JSON(http.StatusOK, config)
}
