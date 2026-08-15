package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

type thirdPartyConnector struct {
	ID         string          `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64          `json:"tenant_id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	BaseURL    string          `json:"base_url"`
	AuthType   string          `json:"auth_type"`
	AuthToken  string          `json:"-"`
	Enabled    bool            `json:"enabled"`
	SyncRule   json.RawMessage `json:"sync_rule" gorm:"type:json"`
	LastSyncAt *time.Time      `json:"last_sync_at"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (thirdPartyConnector) TableName() string { return "third_party_connector" }

type thirdPartyConnectorResponse struct {
	thirdPartyConnector
	AuthConfigured bool `json:"auth_configured"`
}

type thirdPartyConnectorRequest struct {
	Name      string          `json:"name" binding:"required"`
	Type      string          `json:"type" binding:"required,oneof=users departments documents"`
	BaseURL   string          `json:"base_url" binding:"required"`
	AuthType  string          `json:"auth_type" binding:"required,oneof=none bearer basic"`
	AuthToken *string         `json:"auth_token"`
	Enabled   *bool           `json:"enabled"`
	SyncRule  json.RawMessage `json:"sync_rule"`
}

type SDPivotIntegrationHandler struct{ db *gorm.DB }

func NewSDPivotIntegrationHandler(db *gorm.DB) *SDPivotIntegrationHandler {
	return &SDPivotIntegrationHandler{db: db}
}

func (h *SDPivotIntegrationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	integrations := rg.Group("/sdpivot/integrations", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	integrations.GET("", h.List)
	integrations.POST("", h.Create)
	integrations.PUT("/:id", h.Update)
	integrations.DELETE("/:id", h.Delete)
	integrations.POST("/:id/sync/:resource", h.Sync)
	sync := rg.Group("/sdpivot", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	sync.GET("/users/sync", h.SyncUsers)
	sync.GET("/departments/sync", h.SyncDepartments)
	sync.POST("/users/sync", h.BatchSyncUsers)
	sync.POST("/departments/sync", h.BatchSyncDepartments)
	sync.POST("/knowledge/search", h.SearchKnowledge)
}

func (h *SDPivotIntegrationHandler) SyncUsers(c *gin.Context) {
	h.syncConfiguration(c, "users")
}

func (h *SDPivotIntegrationHandler) SyncDepartments(c *gin.Context) {
	h.syncConfiguration(c, "departments")
}

func (h *SDPivotIntegrationHandler) syncConfiguration(c *gin.Context, resource string) {
	var connector thirdPartyConnector
	err := middleware.TenantDB(c, h.db).Where("tenant_id = ? AND type = ? AND enabled = TRUE", middleware.GetTenantID(c), resource).
		Order("updated_at DESC").First(&connector).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, gin.H{"resource": resource, "configured": false, "status": "not_configured", "message": "未配置可用的同步连接器"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load sync configuration"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"resource": resource, "configured": true, "status": "configured", "message": "同步连接器已配置",
		"integration": thirdPartyConnectorResponse{thirdPartyConnector: connector, AuthConfigured: connector.AuthToken != ""},
	})
}

type thirdPartyUserSyncRow struct {
	UserID       string           `json:"userId"`
	Username     string           `json:"username"`
	Name         string           `json:"name"`
	Phone        string           `json:"phone"`
	Email        string           `json:"email"`
	DepartmentID string           `json:"department"`
	Role         types.AccessRole `json:"role"`
	Status       string           `json:"status"`
}

func (h *SDPivotIntegrationHandler) BatchSyncUsers(c *gin.Context) {
	var rows []thirdPartyUserSyncRow
	if err := c.ShouldBindJSON(&rows); err != nil || len(rows) == 0 || len(rows) > maxAdminUserBatchRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("body must contain 1 to %d users", maxAdminUserBatchRows)})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	results := make([]gin.H, 0, len(rows))
	for index, row := range rows {
		result := h.syncUserRow(db, tenantID, row)
		result["row"] = index + 1
		result["user_id"] = strings.TrimSpace(row.UserID)
		results = append(results, result)
	}
	c.JSON(http.StatusOK, gin.H{"total": len(rows), "results": results})
}

func (h *SDPivotIntegrationHandler) syncUserRow(db *gorm.DB, tenantID uint64, row thirdPartyUserSyncRow) gin.H {
	row.UserID = strings.TrimSpace(row.UserID)
	row.Username = strings.TrimSpace(row.Username)
	row.Name = strings.TrimSpace(row.Name)
	row.Phone = normalizeImportPhone(strings.TrimSpace(row.Phone))
	row.Email = strings.ToLower(strings.TrimSpace(row.Email))
	row.DepartmentID = strings.TrimSpace(row.DepartmentID)
	row.Status = strings.ToLower(strings.TrimSpace(row.Status))
	if row.UserID == "" || len(row.UserID) > 36 {
		return gin.H{"status": "failed", "reason": "userId is required and must not exceed 36 characters"}
	}
	if row.Username == "" {
		row.Username = row.UserID
	}
	if row.Email == "" {
		row.Email = row.UserID + "@sdpivot.local"
	}
	if address, err := mail.ParseAddress(row.Email); err != nil || !strings.EqualFold(address.Address, row.Email) {
		return gin.H{"status": "failed", "reason": "invalid email"}
	}
	if row.DepartmentID == "" {
		return gin.H{"status": "failed", "reason": "department is required"}
	}
	var departmentCount int64
	if err := db.Model(&types.Department{}).Where("tenant_id = ? AND id = ?", tenantID, row.DepartmentID).Count(&departmentCount).Error; err != nil || departmentCount != 1 {
		return gin.H{"status": "failed", "reason": "department not found"}
	}
	row.Role = types.NormalizeAccessRole(string(row.Role))
	if !row.Role.IsValid() || row.Role == types.AccessRoleSuperAdmin {
		return gin.H{"status": "failed", "reason": "invalid role"}
	}
	if row.Status == "" {
		row.Status = "enabled"
	}
	if row.Status != "enabled" && row.Status != "disabled" {
		return gin.H{"status": "failed", "reason": "status must be enabled or disabled"}
	}

	var user types.User
	err := db.Unscoped().Where("id = ? AND tenant_id = ?", row.UserID, tenantID).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return gin.H{"status": "failed", "reason": "failed to load user"}
	}
	now := time.Now()
	active := row.Status == "enabled"
	departmentID := row.DepartmentID
	if err == nil {
		updates := map[string]any{"username": row.Username, "email": row.Email, "department_id": departmentID, "access_role": row.Role, "is_active": active, "deleted_at": nil, "updated_at": now}
		profileUpdates := map[string]any{"nickname": firstNonEmpty(row.Name, row.Username), "status": map[bool]string{true: "active", false: "disabled"}[active], "updated_at": now, "deleted_at": nil}
		if row.Phone != "" {
			profileUpdates["phone"] = row.Phone
		}
		updateErr := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Unscoped().Model(&user).Updates(updates).Error; err != nil {
				return err
			}
			memberStatus := types.TenantMemberStatusActive
			if !active {
				memberStatus = types.TenantMemberStatusSuspended
			}
			if err := tx.Model(&types.TenantMember{}).Where("tenant_id = ? AND user_id = ?", tenantID, user.ID).
				Updates(map[string]any{"status": memberStatus, "updated_at": now}).Error; err != nil {
				return err
			}
			var profile types.SDPivotUserProfile
			profileErr := tx.Unscoped().Where("user_id = ?", user.ID).First(&profile).Error
			if profileErr == nil {
				return tx.Unscoped().Model(&profile).Updates(profileUpdates).Error
			}
			if !errors.Is(profileErr, gorm.ErrRecordNotFound) {
				return profileErr
			}
			phone := nullableString(row.Phone)
			return tx.Create(&types.SDPivotUserProfile{UserID: user.ID, Phone: phone, Nickname: firstNonEmpty(row.Name, row.Username), Status: profileUpdates["status"].(string), CreatedAt: now, UpdatedAt: now}).Error
		})
		if updateErr != nil {
			return gin.H{"status": "failed", "reason": userImportDatabaseError(updateErr)}
		}
		status := "updated"
		if !active {
			status = "disabled"
		}
		return gin.H{"status": status}
	}

	password, passwordErr := generateInitialPassword()
	if passwordErr != nil {
		return gin.H{"status": "failed", "reason": "failed to generate initial password"}
	}
	hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if hashErr != nil {
		return gin.H{"status": "failed", "reason": "failed to secure initial password"}
	}
	user = types.User{ID: row.UserID, Username: row.Username, Email: row.Email, PasswordHash: string(hash), TenantID: tenantID, IsActive: active, AccessRole: row.Role, DepartmentID: &departmentID, MustChangePassword: true, CreatedAt: now, UpdatedAt: now}
	phone := nullableString(row.Phone)
	if createErr := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		memberStatus := types.TenantMemberStatusActive
		if !active {
			memberStatus = types.TenantMemberStatusSuspended
		}
		if err := tx.Create(&types.TenantMember{UserID: user.ID, TenantID: tenantID, Role: types.TenantRoleViewer, Status: memberStatus, JoinedAt: now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			return err
		}
		profileStatus := "active"
		if !active {
			profileStatus = "disabled"
		}
		return tx.Create(&types.SDPivotUserProfile{UserID: user.ID, Phone: phone, Nickname: firstNonEmpty(row.Name, row.Username), Status: profileStatus, CreatedAt: now, UpdatedAt: now}).Error
	}); createErr != nil {
		return gin.H{"status": "failed", "reason": userImportDatabaseError(createErr)}
	}
	return gin.H{"status": "created"}
}

type thirdPartyDepartmentSyncRow struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
	Sort     int    `json:"sort"`
}

func (h *SDPivotIntegrationHandler) BatchSyncDepartments(c *gin.Context) {
	var rows []thirdPartyDepartmentSyncRow
	if err := c.ShouldBindJSON(&rows); err != nil || len(rows) == 0 || len(rows) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body must contain 1 to 1000 departments"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	results := make([]gin.H, 0, len(rows))
	for index, row := range rows {
		row.ID = strings.TrimSpace(row.ID)
		row.Name = strings.TrimSpace(row.Name)
		row.ParentID = strings.TrimSpace(row.ParentID)
		result := gin.H{"row": index + 1, "department_id": row.ID}
		if row.ID == "" || len(row.ID) > 36 || row.Name == "" || len(row.ParentID) > 36 || row.ParentID == row.ID {
			result["status"] = "failed"
			result["reason"] = "invalid id, name, or parent_id"
			results = append(results, result)
			continue
		}
		var department types.Department
		err := db.Unscoped().Where("tenant_id = ? AND id = ?", tenantID, row.ID).First(&department).Error
		now := time.Now()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			department = types.Department{ID: row.ID, TenantID: tenantID, ParentID: row.ParentID, Name: row.Name, SortOrder: row.Sort, CreatedAt: now, UpdatedAt: now}
			if err := db.Create(&department).Error; err != nil {
				result["status"] = "failed"
				result["reason"] = err.Error()
			} else {
				result["status"] = "created"
			}
		} else if err != nil {
			result["status"] = "failed"
			result["reason"] = "failed to load department"
		} else if err := db.Unscoped().Model(&department).Updates(map[string]any{"name": row.Name, "parent_id": row.ParentID, "sort_order": row.Sort, "deleted_at": nil, "updated_at": now}).Error; err != nil {
			result["status"] = "failed"
			result["reason"] = err.Error()
		} else {
			result["status"] = "updated"
		}
		results = append(results, result)
	}
	c.JSON(http.StatusOK, gin.H{"total": len(rows), "results": results})
}

func (h *SDPivotIntegrationHandler) SearchKnowledge(c *gin.Context) {
	var req struct {
		Query    string   `json:"query" binding:"required"`
		SpaceIDs []string `json:"space_ids"`
		TopK     int      `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Query) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}
	if req.TopK < 1 {
		req.TopK = 10
	}
	if req.TopK > 100 {
		req.TopK = 100
	}
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	query := strings.TrimSpace(req.Query)
	search := "%" + escapeILike(query) + "%"
	base := db.Table("document_chunks chunks").
		Joins("JOIN documents doc ON doc.id = chunks.document_id AND doc.tenant_id = chunks.tenant_id").
		Joins("JOIN knowledge_spaces space ON space.id = doc.space_id AND space.tenant_id = doc.tenant_id").
		Where("chunks.tenant_id = ? AND doc.deleted_at IS NULL AND space.deleted_at IS NULL AND doc.parse_status = ?", tenantID, "completed").
		Where("doc.space_id IN (?)", visibleSpaceIDsQuery(db, tenantID, middleware.GetUserID(c))).
		Where("chunks.content ILIKE ?", search)
	if spaceIDs := uniqueStrings(req.SpaceIDs); len(spaceIDs) > 0 {
		base = base.Where("doc.space_id IN ?", spaceIDs)
	}
	type hit struct {
		ChunkID       string  `json:"chunk_id"`
		DocumentID    string  `json:"document_id"`
		DocumentTitle string  `json:"document_title"`
		Text          string  `json:"text"`
		SpaceID       string  `json:"space_id"`
		SpaceName     string  `json:"space_name"`
		Similarity    float64 `json:"similarity"`
	}
	hits := make([]hit, 0)
	if err := base.Select("chunks.id AS chunk_id, doc.id AS document_id, doc.title AS document_title, chunks.content AS text, space.id AS space_id, space.name AS space_name, 1.0 AS similarity").
		Order("doc.updated_at DESC, chunks.chunk_index ASC").Limit(req.TopK).Scan(&hits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search knowledge"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"query": query, "hits": hits, "total": len(hits)})
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (h *SDPivotIntegrationHandler) List(c *gin.Context) {
	rows := make([]thirdPartyConnector, 0)
	if err := middleware.TenantDB(c, h.db).Where("tenant_id = ?", middleware.GetTenantID(c)).Order("created_at DESC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load integrations"})
		return
	}
	items := make([]thirdPartyConnectorResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, thirdPartyConnectorResponse{thirdPartyConnector: row, AuthConfigured: row.AuthToken != ""})
	}
	c.JSON(http.StatusOK, gin.H{"integrations": items})
}

func (h *SDPivotIntegrationHandler) Create(c *gin.Context) {
	req, ok := bindConnectorRequest(c)
	if !ok {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now()
	row := thirdPartyConnector{
		ID: uuid.NewString(), TenantID: middleware.GetTenantID(c), Name: strings.TrimSpace(req.Name),
		Type: req.Type, BaseURL: strings.TrimRight(strings.TrimSpace(req.BaseURL), "/"), AuthType: req.AuthType,
		Enabled: enabled, SyncRule: normalizedSyncRule(req.SyncRule), CreatedAt: now, UpdatedAt: now,
	}
	if req.AuthToken != nil {
		row.AuthToken = strings.TrimSpace(*req.AuthToken)
	}
	if err := middleware.TenantDB(c, h.db).Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create integration"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"integration": thirdPartyConnectorResponse{thirdPartyConnector: row, AuthConfigured: row.AuthToken != ""}})
}

func (h *SDPivotIntegrationHandler) Update(c *gin.Context) {
	req, ok := bindConnectorRequest(c)
	if !ok {
		return
	}
	db := middleware.TenantDB(c, h.db)
	var row thirdPartyConnector
	if err := db.Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load integration"})
		}
		return
	}
	updates := map[string]interface{}{
		"name": strings.TrimSpace(req.Name), "type": req.Type,
		"base_url": strings.TrimRight(strings.TrimSpace(req.BaseURL), "/"), "auth_type": req.AuthType,
		"sync_rule": normalizedSyncRule(req.SyncRule), "updated_at": time.Now(),
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.AuthToken != nil && strings.TrimSpace(*req.AuthToken) != "" {
		updates["auth_token"] = strings.TrimSpace(*req.AuthToken)
	}
	if req.AuthType == "none" {
		updates["auth_token"] = ""
	}
	if err := db.Model(&row).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update integration"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "integration updated"})
}

func (h *SDPivotIntegrationHandler) Delete(c *gin.Context) {
	result := middleware.TenantDB(c, h.db).Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).Delete(&thirdPartyConnector{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete integration"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "integration deleted"})
}

func (h *SDPivotIntegrationHandler) Sync(c *gin.Context) {
	resource := c.Param("resource")
	if resource != "users" && resource != "departments" && resource != "documents" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported sync resource"})
		return
	}
	var count int64
	if err := middleware.TenantDB(c, h.db).Model(&thirdPartyConnector{}).
		Where("id = ? AND tenant_id = ?", c.Param("id"), middleware.GetTenantID(c)).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load integration"})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "integration not found"})
		return
	}
	// TODO: enqueue the connector-specific synchronization worker.
	c.JSON(http.StatusOK, gin.H{"message": "同步任务已排队（暂未实现）", "status": "queued", "resource": resource})
}

func bindConnectorRequest(c *gin.Context) (thirdPartyConnectorRequest, bool) {
	var req thirdPartyConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return req, false
	}
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	if err := secutils.ValidateURLForSSRF(req.BaseURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or unsafe connector base_url"})
		return req, false
	}
	if len(req.SyncRule) > 0 && string(req.SyncRule) != "null" && !json.Valid(req.SyncRule) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sync_rule must be valid JSON"})
		return req, false
	}
	return req, true
}

func normalizedSyncRule(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage([]byte("{}"))
	}
	return raw
}
