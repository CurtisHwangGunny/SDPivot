package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================
// Filters — 敏感词过滤（PRD §4.1.2）
// ============================================================

// SensitiveWord 敏感词条目
type OpsSensitiveWord struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Word      string    `json:"word" gorm:"type:varchar(200);not null;index"`
	Category  string    `json:"category" gorm:"type:varchar(50);not null;default:general"`
	Status    string    `json:"status" gorm:"type:varchar(20);not null;default:active"`
	CreatedBy string    `json:"created_by" gorm:"type:varchar(36)"`
	CreatedAt time.Time `json:"created_at"`
}

func (OpsSensitiveWord) TableName() string { return "sensitive_words" }

func (h *SmartKnoraOpsAdminHandler) ListSensitiveWords(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	page, pageSize := parsePagination(c)
	search := c.Query("search")
	category := c.Query("category")

	q := h.db.Model(&OpsSensitiveWord{})
	if search != "" {
		q = q.Where("word ILIKE ?", "%"+search+"%")
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}

	var total int64
	q.Count(&total)

	var words []OpsSensitiveWord
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&words)

	c.JSON(http.StatusOK, gin.H{"words": words, "total": total, "page": page, "page_size": pageSize})
}

func (h *SmartKnoraOpsAdminHandler) CreateSensitiveWord(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	userID := getUserIDFromCtx(c)

	var req struct {
		Word     string `json:"word" binding:"required"`
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Category == "" {
		req.Category = "general"
	}

	sw := OpsSensitiveWord{
		ID:        uuid.New().String(),
		Word:      req.Word,
		Category:  req.Category,
		Status:    "active",
		CreatedBy: userID,
		CreatedAt: time.Now(),
	}
	if err := h.db.Create(&sw).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create"})
		return
	}
	h.writeAuditLog(c, "create_sensitive_word", "sensitive_word", sw.ID, "word: "+req.Word)
	c.JSON(http.StatusCreated, gin.H{"word": sw})
}

func (h *SmartKnoraOpsAdminHandler) DeleteSensitiveWord(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")
	h.db.Where("id = ?", id).Delete(&OpsSensitiveWord{})
	h.writeAuditLog(c, "delete_sensitive_word", "sensitive_word", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *SmartKnoraOpsAdminHandler) ListFilterHits(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	page, pageSize := parsePagination(c)

	// 查找 org_ext 中 auth_status=suspended 或标记待处理的企业
	type HitRow struct {
		OrgID       string     `json:"org_id"`
		OrgName     string     `json:"org_name"`
		AuthStatus  string     `json:"auth_status"`
		SubStatus   string     `json:"subscription_status"`
		CreatedAt   time.Time  `json:"created_at"`
		OwnerID     string     `json:"owner_id"`
	}

	var total int64
	h.db.Table("organizations o").
		Joins("LEFT JOIN org_ext e ON e.org_id = o.id").
		Where("o.deleted_at IS NULL AND (e.auth_status = 'suspended' OR e.auth_status = 'flagged')").
		Count(&total)

	var rows []HitRow
	h.db.Table("organizations o").
		Select("o.id as org_id, o.name as org_name, COALESCE(e.auth_status, 'trial') as auth_status, COALESCE(e.subscription_status, 'free') as subscription_status, o.created_at, o.owner_id").
		Joins("LEFT JOIN org_ext e ON e.org_id = o.id").
		Where("o.deleted_at IS NULL AND (e.auth_status = 'suspended' OR e.auth_status = 'flagged')").
		Order("o.created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows)

	c.JSON(http.StatusOK, gin.H{"hits": rows, "total": total, "page": page, "page_size": pageSize})
}

func (h *SmartKnoraOpsAdminHandler) UpdateFilterHit(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	orgID := c.Param("orgId")

	var req struct {
		Action string `json:"action" binding:"required,oneof=freeze release"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := "active"
	if req.Action == "freeze" {
		status = "suspended"
	}

	h.db.Table("org_ext").Where("org_id = ?", orgID).Update("auth_status", status)
	h.writeAuditLog(c, "filter_"+req.Action, "enterprise", orgID, "status: "+status)

	c.JSON(http.StatusOK, gin.H{"message": "updated", "status": status})
}

// ============================================================
// Billing — 计费管理（PRD §4.1.3）
// ============================================================

type OpsBillingPlan struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name         string    `json:"name" gorm:"type:varchar(100);not null"`
	Price        float64   `json:"price" gorm:"type:decimal(10,2);not null;default:0"`
	TokenQuota   int64     `json:"token_quota" gorm:"not null;default:0"`
	StorageQuota int64     `json:"storage_quota" gorm:"not null;default:0"`
	Features     string    `json:"features" gorm:"type:jsonb;not null;default:'{}'"`
	Status       string    `json:"status" gorm:"type:varchar(20);not null;default:active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (OpsBillingPlan) TableName() string { return "billing_plans" }

type OpsEnterpriseSubscription struct {
	ID        string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	OrgID     string     `json:"org_id" gorm:"type:varchar(36);not null;index"`
	PlanID    string     `json:"plan_id" gorm:"type:varchar(36);not null"`
	Status    string     `json:"status" gorm:"type:varchar(20);not null;default:active"`
	StartedAt time.Time  `json:"started_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (OpsEnterpriseSubscription) TableName() string { return "enterprise_subscriptions" }

type OpsInvoice struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	OrgID        string    `json:"org_id" gorm:"type:varchar(36);not null;index"`
	PlanID       string    `json:"plan_id" gorm:"type:varchar(36)"`
	Amount       float64   `json:"amount" gorm:"type:decimal(10,2);not null;default:0"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	Status       string    `json:"status" gorm:"type:varchar(20);not null;default:pending"`
	CreatedAt    time.Time `json:"created_at"`
}

func (OpsInvoice) TableName() string { return "invoices" }

func (h *SmartKnoraOpsAdminHandler) ListBillingPlans(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	var plans []OpsBillingPlan
	h.db.Order("created_at DESC").Find(&plans)
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

func (h *SmartKnoraOpsAdminHandler) CreateBillingPlan(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	var req struct {
		Name         string  `json:"name" binding:"required"`
		Price        float64 `json:"price"`
		TokenQuota   int64   `json:"token_quota"`
		StorageQuota int64   `json:"storage_quota"`
		Features     string  `json:"features"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Features == "" {
		req.Features = "{}"
	}
	plan := OpsBillingPlan{
		ID: uuid.New().String(), Name: req.Name, Price: req.Price,
		TokenQuota: req.TokenQuota, StorageQuota: req.StorageQuota,
		Features: req.Features, Status: "active",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	h.db.Create(&plan)
	h.writeAuditLog(c, "create_billing_plan", "billing_plan", plan.ID, "name: "+req.Name)
	c.JSON(http.StatusCreated, gin.H{"plan": plan})
}

func (h *SmartKnoraOpsAdminHandler) UpdateBillingPlan(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")
	var req struct {
		Name         *string  `json:"name"`
		Price        *float64 `json:"price"`
		TokenQuota   *int64   `json:"token_quota"`
		StorageQuota *int64   `json:"storage_quota"`
		Status       *string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != nil { updates["name"] = *req.Name }
	if req.Price != nil { updates["price"] = *req.Price }
	if req.TokenQuota != nil { updates["token_quota"] = *req.TokenQuota }
	if req.StorageQuota != nil { updates["storage_quota"] = *req.StorageQuota }
	if req.Status != nil { updates["status"] = *req.Status }
	h.db.Model(&OpsBillingPlan{}).Where("id = ?", id).Updates(updates)
	h.writeAuditLog(c, "update_billing_plan", "billing_plan", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *SmartKnoraOpsAdminHandler) DeleteBillingPlan(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")
	h.db.Where("id = ?", id).Delete(&OpsBillingPlan{})
	h.writeAuditLog(c, "delete_billing_plan", "billing_plan", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *SmartKnoraOpsAdminHandler) ListSubscriptions(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")
	page, pageSize := parsePagination(c)

	type SubRow struct {
		ID        string     `json:"id"`
		OrgID     string     `json:"org_id"`
		OrgName   string     `json:"org_name"`
		PlanID    string     `json:"plan_id"`
		PlanName  string     `json:"plan_name"`
		Status    string     `json:"status"`
		StartedAt time.Time  `json:"started_at"`
		ExpiresAt *time.Time `json:"expires_at"`
	}

	var total int64
	h.db.Table("enterprise_subscriptions es").
		Joins("LEFT JOIN organizations o ON o.id = es.org_id").
		Joins("LEFT JOIN billing_plans bp ON bp.id = es.plan_id").
		Count(&total)

	var rows []SubRow
	h.db.Table("enterprise_subscriptions es").
		Select("es.id, es.org_id, COALESCE(o.name, '') as org_name, es.plan_id, COALESCE(bp.name, '') as plan_name, es.status, es.started_at, es.expires_at").
		Joins("LEFT JOIN organizations o ON o.id = es.org_id").
		Joins("LEFT JOIN billing_plans bp ON bp.id = es.plan_id").
		Order("es.created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows)

	c.JSON(http.StatusOK, gin.H{"subscriptions": rows, "total": total, "page": page, "page_size": pageSize})
}

func (h *SmartKnoraOpsAdminHandler) UpdateSubscription(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	orgID := c.Param("orgId")
	var req struct {
		PlanID    string  `json:"plan_id" binding:"required"`
		ExpiresAt *string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Upsert subscription
	var existing OpsEnterpriseSubscription
	if h.db.Where("org_id = ?", orgID).First(&existing).Error == nil {
		h.db.Model(&existing).Updates(map[string]interface{}{
			"plan_id": req.PlanID, "status": "active", "updated_at": time.Now(),
		})
	} else {
		sub := OpsEnterpriseSubscription{
			ID: uuid.New().String(), OrgID: orgID, PlanID: req.PlanID,
			Status: "active", StartedAt: time.Now(),
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		h.db.Create(&sub)
	}

	// Sync org_ext subscription_status
	var plan OpsBillingPlan
	planName := "free"
	if h.db.Where("id = ?", req.PlanID).First(&plan).Error == nil {
		planName = plan.Name
	}
	h.db.Table("org_ext").Where("org_id = ?", orgID).Update("subscription_status", planName)

	h.writeAuditLog(c, "update_subscription", "enterprise", orgID, "plan: "+req.PlanID)
	c.JSON(http.StatusOK, gin.H{"message": "subscription updated"})
}

func (h *SmartKnoraOpsAdminHandler) ListInvoices(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")
	page, pageSize := parsePagination(c)

	var total int64
	h.db.Table("invoices").Count(&total)

	var invoices []OpsInvoice
	h.db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&invoices)

	c.JSON(http.StatusOK, gin.H{"invoices": invoices, "total": total, "page": page, "page_size": pageSize})
}

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

func (h *SmartKnoraOpsAdminHandler) ListConfigs(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	var configs []OpsSystemConfig
	h.db.Order("key ASC").Find(&configs)
	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

func (h *SmartKnoraOpsAdminHandler) UpdateConfig(c *gin.Context) {
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

	h.writeAuditLog(c, "update_config", "system_config", key, "value: "+req.Value)
	c.JSON(http.StatusOK, gin.H{"message": "config updated", "key": key, "value": req.Value})
}

func (h *SmartKnoraOpsAdminHandler) GetTrialConfig(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	var cfg OpsSystemConfig
	trialDays := "30"
	if h.db.Where("key = ?", "trial_days").First(&cfg).Error == nil {
		trialDays = cfg.Value
	}
	extendedDays := "90"
	if h.db.Where("key = ?", "extended_trial_days").First(&cfg).Error == nil {
		extendedDays = cfg.Value
	}
	downgradeSpaceLimit := "1"
	if h.db.Where("key = ?", "downgrade_space_limit").First(&cfg).Error == nil {
		downgradeSpaceLimit = cfg.Value
	}
	c.JSON(http.StatusOK, gin.H{
		"trial_days":             trialDays,
		"extended_trial_days":    extendedDays,
		"downgrade_space_limit":  downgradeSpaceLimit,
	})
}

func (h *SmartKnoraOpsAdminHandler) UpdateTrialConfig(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	var req struct {
		TrialDays           int `json:"trial_days"`
		ExtendedTrialDays   int `json:"extended_trial_days"`
		DowngradeSpaceLimit int `json:"downgrade_space_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.upsertConfig("trial_days", strconv.Itoa(req.TrialDays), "试用期天数")
	h.upsertConfig("extended_trial_days", strconv.Itoa(req.ExtendedTrialDays), "认证后延长天数")
	h.upsertConfig("downgrade_space_limit", strconv.Itoa(req.DowngradeSpaceLimit), "降级后空间数量限制")

	h.writeAuditLog(c, "update_trial_config", "system_config", "trial", "")
	c.JSON(http.StatusOK, gin.H{"message": "trial config updated"})
}

func (h *SmartKnoraOpsAdminHandler) upsertConfig(key, value, description string) {
	var existing OpsSystemConfig
	if h.db.Where("key = ?", key).First(&existing).Error == nil {
		h.db.Model(&existing).Updates(map[string]interface{}{"value": value, "updated_at": time.Now()})
	} else {
		h.db.Create(&OpsSystemConfig{Key: key, Value: value, Description: description, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	}
}

// ============================================================
// Models — 模型管理（PRD §4.1.3）
// ============================================================

func (h *SmartKnoraOpsAdminHandler) ListModels(c *gin.Context) {
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

func (h *SmartKnoraOpsAdminHandler) CreateModel(c *gin.Context) {
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

func (h *SmartKnoraOpsAdminHandler) UpdateModel(c *gin.Context) {
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
	if req.DisplayName != nil { updates["display_name"] = *req.DisplayName }
	if req.Source != nil { updates["source"] = *req.Source }
	if req.Description != nil { updates["description"] = *req.Description }
	if req.Status != nil { updates["status"] = *req.Status }
	h.db.Table("models").Where("id = ?", id).Updates(updates)
	h.writeAuditLog(c, "update_model", "model", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "model updated"})
}

func (h *SmartKnoraOpsAdminHandler) DeleteModel(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")
	h.db.Table("models").Where("id = ? AND is_builtin = false", id).Update("deleted_at", time.Now())
	h.writeAuditLog(c, "delete_model", "model", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "model deleted"})
}

func (h *SmartKnoraOpsAdminHandler) SetDefaultModel(c *gin.Context) {
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

// Ensure unused imports are referenced
var _ = fmt.Sprintf
var _ = gorm.ErrRecordNotFound
