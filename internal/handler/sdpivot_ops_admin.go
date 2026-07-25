package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SDPivotOpsAdminHandler handles operations admin management endpoints.
// All endpoints require ops_admin role (PRD §0.1.4).
type SDPivotOpsAdminHandler struct {
	db *gorm.DB
}

func NewSDPivotOpsAdminHandler(db *gorm.DB) *SDPivotOpsAdminHandler {
	return &SDPivotOpsAdminHandler{db: db}
}

// RegisterOpsRoutes registers protected ops admin routes (requires ops_admin role).
func (h *SDPivotOpsAdminHandler) RegisterOpsRoutes(rg *gin.RouterGroup) {
	ops := rg.Group("/ops")
	{
		ops.GET("/dashboard", h.GetOpsDashboard)
		ops.GET("/enterprises", h.ListEnterprises)
		ops.GET("/enterprises/:id", h.GetEnterprise)
		ops.PUT("/enterprises/:id/status", h.UpdateEnterpriseStatus)
		ops.GET("/tenants", h.ListEnterprises) // compat alias
		ops.GET("/users", h.ListUsers)
		ops.PUT("/users/:id/status", h.UpdateUserStatus)
		ops.GET("/audit-logs", h.GetAuditLogs)
		ops.GET("/audit-logs/export", h.ExportAuditLogs)
		ops.GET("/audit-log", h.GetAuditLogs) // compat alias
		ops.POST("/announcements", h.CreateAnnouncement)
		ops.GET("/announcements", h.ListAnnouncements)
		ops.DELETE("/announcements/:id", h.DeleteAnnouncement)

		// Billing (PRD 4.1.3)
		ops.GET("/billing/plans", h.ListBillingPlans)
		ops.POST("/billing/plans", h.CreateBillingPlan)
		ops.PUT("/billing/plans/:id", h.UpdateBillingPlan)
		ops.DELETE("/billing/plans/:id", h.DeleteBillingPlan)
		ops.GET("/billing/subscriptions", h.ListSubscriptions)
		ops.PUT("/billing/subscriptions/:orgId", h.UpdateSubscription)
		ops.GET("/billing/invoices", h.ListInvoices)

		// Config (PRD 4.1.3)
		ops.GET("/config", h.ListConfigs)
		ops.GET("/config/trial", h.GetTrialConfig)
		ops.PUT("/config/trial", h.UpdateTrialConfig)
		ops.PUT("/config/:key", h.UpdateConfig)

		// Models (PRD 4.1.3)
		ops.GET("/models", h.ListModels)
		ops.POST("/models", h.CreateModel)
		ops.PUT("/models/:id", h.UpdateModel)
		ops.DELETE("/models/:id", h.DeleteModel)
		ops.PUT("/models/:id/default", h.SetDefaultModel)
	}
}

// RegisterPublicOpsRoutes registers public ops routes (no auth required).
func (h *SDPivotOpsAdminHandler) RegisterPublicOpsRoutes(rg *gin.RouterGroup) {
	ops := rg.Group("/ops")
	ops.GET("/announcements/active", h.GetActiveAnnouncements)
}

// requireOpsAdmin checks if current user has ops_admin role.
func requireOpsAdmin(c *gin.Context) bool {
	role, _ := c.Get("role")
	return role == "ops_admin"
}

func denyIfNotOpsAdmin(c *gin.Context) bool {
	if !requireOpsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "ops_admin access required"})
		return true
	}
	return false
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

// ── Dashboard ──────────────────────────────────────────────

func (h *SDPivotOpsAdminHandler) GetOpsDashboard(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	var tenantCount, userCount, docCount, tokenTotal int64
	h.db.Table("organizations").Where("deleted_at IS NULL").Count(&tenantCount)
	h.db.Table("users").Where("deleted_at IS NULL").Count(&userCount)
	h.db.Table("documents").Where("deleted_at IS NULL").Count(&docCount)
	h.db.Table("token_usage").Count(&tokenTotal)

	var todayToken int64
	today := time.Now().Truncate(24 * time.Hour)
	h.db.Table("token_usage").Where("created_at >= ?", today).Count(&todayToken)

	var storageBytes int64
	h.db.Table("documents").Where("deleted_at IS NULL").Select("COALESCE(SUM(file_size), 0)").Row().Scan(&storageBytes)

	c.JSON(http.StatusOK, gin.H{
		"tenant_count":      tenantCount,
		"user_count":        userCount,
		"document_count":    docCount,
		"token_usage_total": tokenTotal,
		"token_usage_today": todayToken,
		"storage_bytes":     storageBytes,
	})
}

// ── Enterprises ────────────────────────────────────────────

func (h *SDPivotOpsAdminHandler) ListEnterprises(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	page, pageSize := parsePagination(c)
	search := c.Query("search")

	q := h.db.Table("organizations").Where("deleted_at IS NULL")
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}

	var total int64
	q.Count(&total)

	type EnterpriseRow struct {
		ID          string     `json:"id"`
		Name        string     `json:"name"`
		Description string     `json:"description"`
		OwnerID     string     `json:"owner_id"`
		InviteCode  string     `json:"invite_code"`
		MemberCount int64      `json:"member_count"`
		AuthStatus  string     `json:"auth_status"`
		ExpiresAt   *time.Time `json:"expires_at"`
		CreatedAt   time.Time  `json:"created_at"`
	}

	var rows []EnterpriseRow
	rawSQL := `
		SELECT o.id, o.name, o.description, o.owner_id, o.invite_code,
		       COALESCE(cnt.mc, 0) AS member_count,
		       COALESCE(e.auth_status, 'trial') AS auth_status,
		       e.auth_expires_at AS expires_at,
		       o.created_at
		FROM organizations o
		LEFT JOIN (SELECT org_id, COUNT(*) AS mc FROM org_members WHERE status = 'active' GROUP BY org_id) cnt ON cnt.org_id = o.id
		LEFT JOIN org_ext e ON e.org_id = o.id
		WHERE o.deleted_at IS NULL
	`
	args := []interface{}{}
	if search != "" {
		rawSQL += " AND o.name ILIKE ?"
		args = append(args, "%"+search+"%")
	}
	rawSQL += `
		ORDER BY o.created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, pageSize, (page-1)*pageSize)
	h.db.Raw(rawSQL, args...).Scan(&rows)

	c.JSON(http.StatusOK, gin.H{
		"enterprises": rows,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

func (h *SDPivotOpsAdminHandler) GetEnterprise(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	id := c.Param("id")
	var org types.Organization
	if err := h.db.Where("id = ?", id).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "enterprise not found"})
		return
	}

	var ext types.OrgExt
	h.db.Where("org_id = ?", id).First(&ext)

	var memberCount int64
	h.db.Table("org_members").Where("org_id = ? AND status = 'active'", id).Count(&memberCount)

	var members []types.SDPivotOrgMember
	h.db.Where("org_id = ?", id).Find(&members)

	c.JSON(http.StatusOK, gin.H{
		"enterprise":   org,
		"auth_status":  ext.AuthStatus,
		"expires_at":   ext.AuthExpiresAt,
		"member_count": memberCount,
		"members":      members,
	})
}

func (h *SDPivotOpsAdminHandler) UpdateEnterpriseStatus(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required,oneof=active suspended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{"auth_status": req.Status, "updated_at": time.Now()}
	h.db.Table("org_ext").Where("org_id = ?", id).Updates(updates)

	// Write audit log
	h.writeAuditLog(c, "update_enterprise_status", "enterprise", id, "status: "+req.Status)

	c.JSON(http.StatusOK, gin.H{"message": "enterprise status updated", "status": req.Status})
}

// ── Users ──────────────────────────────────────────────────

func (h *SDPivotOpsAdminHandler) ListUsers(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	page, pageSize := parsePagination(c)
	search := c.Query("search")
	activeFilter := c.Query("is_active")

	type UserRow struct {
		ID         string    `json:"id"`
		Username   string    `json:"username"`
		Email      string    `json:"email"`
		Phone      string    `json:"phone"`
		Nickname   string    `json:"nickname"`
		IsActive   bool      `json:"is_active"`
		IsOpsAdmin bool      `json:"is_ops_admin" gorm:"column:is_ops_admin"`
		TenantID   uint64    `json:"tenant_id"`
		CreatedAt  time.Time `json:"created_at"`
	}

	q := h.db.Table("users").Select("users.id, users.username, users.email, users.is_active, users.is_ops_admin, users.tenant_id, users.created_at, COALESCE(p.phone, '') AS phone, COALESCE(p.nickname, '') AS nickname").
		Joins("LEFT JOIN smartknora_user_profiles p ON p.user_id = users.id").
		Where("users.deleted_at IS NULL")

	if search != "" {
		q = q.Where("users.username ILIKE ? OR users.email ILIKE ? OR p.phone ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if activeFilter == "true" {
		q = q.Where("users.is_active = true")
	} else if activeFilter == "false" {
		q = q.Where("users.is_active = false")
	}

	var total int64
	q.Count(&total)

	var rows []UserRow
	q.Order("users.created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows)

	c.JSON(http.StatusOK, gin.H{
		"users":     rows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *SDPivotOpsAdminHandler) UpdateUserStatus(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	userID := c.Param("id")

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.db.Table("users").Where("id = ?", userID).Update("is_active", req.IsActive)

	h.writeAuditLog(c, "update_user_status", "user", userID, fmt.Sprintf("is_active: %v", req.IsActive))

	c.JSON(http.StatusOK, gin.H{"message": "user status updated", "is_active": req.IsActive})
}

// ── Audit Logs ─────────────────────────────────────────────

func (h *SDPivotOpsAdminHandler) GetAuditLogs(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	page, pageSize := parsePagination(c)
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	userID := c.Query("user_id")
	action := c.Query("action")

	q := h.db.Table("audit_logs").Select(`
		id, tenant_id,
		COALESCE(user_id, actor_user_id) AS user_id,
		COALESCE(username, ''::text) AS username, action,
		COALESCE(resource, target_type) AS resource,
		COALESCE(resource_id, target_id) AS resource_id,
		COALESCE(detail, details->>'detail', ''::text) AS detail,
		COALESCE(ip, ''::text) AS ip, created_at`)
	if startTime != "" {
		q = q.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		q = q.Where("created_at <= ?", endTime)
	}
	if userID != "" {
		q = q.Where("COALESCE(user_id, actor_user_id) = ?", userID)
	}
	if action != "" {
		q = q.Where("action ILIKE ?", "%"+action+"%")
	}

	var total int64
	q.Count(&total)

	type AuditRow struct {
		ID         int64     `json:"id"`
		TenantID   uint64    `json:"tenant_id"`
		UserID     string    `json:"user_id"`
		Username   string    `json:"username"`
		Action     string    `json:"action"`
		Resource   string    `json:"resource"`
		ResourceID string    `json:"resource_id"`
		Detail     string    `json:"detail"`
		IP         string    `json:"ip"`
		CreatedAt  time.Time `json:"created_at"`
	}

	rows := make([]AuditRow, 0)
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows)

	c.JSON(http.StatusOK, gin.H{
		"logs":      rows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *SDPivotOpsAdminHandler) ExportAuditLogs(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	h.db.Exec("SET LOCAL row_security = off")

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"ID", "Time", "User ID", "Username", "Action", "Resource", "Resource ID", "Detail", "IP"})

	type AuditRow struct {
		ID         int64
		UserID     string
		Username   string
		Action     string
		Resource   string
		ResourceID string
		Detail     string
		IP         string
		CreatedAt  time.Time
	}

	rows := make([]AuditRow, 0)
	h.db.Table("audit_logs").Select(`
		id,
		COALESCE(user_id, actor_user_id) AS user_id,
		COALESCE(username, ''::text) AS username, action,
		COALESCE(resource, target_type) AS resource,
		COALESCE(resource_id, target_id) AS resource_id,
		COALESCE(detail, details->>'detail', ''::text) AS detail,
		COALESCE(ip, ''::text) AS ip, created_at`).Order("created_at DESC").Limit(10000).Scan(&rows)

	for _, r := range rows {
		writer.Write([]string{
			strconv.FormatInt(r.ID, 10),
			r.CreatedAt.Format(time.RFC3339),
			r.UserID,
			r.Username,
			r.Action,
			r.Resource,
			r.ResourceID,
			r.Detail,
			r.IP,
		})
	}
	writer.Flush()
}

// ── Announcements ──────────────────────────────────────────

func (h *SDPivotOpsAdminHandler) CreateAnnouncement(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ann := types.Announcement{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Title:     req.Title,
		Content:   req.Content,
		Status:    "published",
		CreatedBy: userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	h.db.Create(&ann)
	c.JSON(http.StatusCreated, gin.H{"announcement": ann})
}

func (h *SDPivotOpsAdminHandler) ListAnnouncements(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	tenantID := middleware.GetTenantID(c)

	var list []types.Announcement
	h.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(50).Find(&list)
	c.JSON(http.StatusOK, gin.H{"announcements": list})
}

func (h *SDPivotOpsAdminHandler) DeleteAnnouncement(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}
	id := c.Param("id")
	h.db.Where("id = ?", id).Delete(&types.Announcement{})
	c.JSON(http.StatusOK, gin.H{"message": "announcement deleted"})
}

// GetActiveAnnouncements returns published announcements (public, no auth).
func (h *SDPivotOpsAdminHandler) GetActiveAnnouncements(c *gin.Context) {
	var list []types.Announcement
	h.db.Where("status = ?", "published").Order("created_at DESC").Limit(20).Find(&list)
	c.JSON(http.StatusOK, gin.H{"announcements": list})
}

// ── Audit Log Helper ───────────────────────────────────────

func (h *SDPivotOpsAdminHandler) writeAuditLog(c *gin.Context, action, resource, resourceID, detail string) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	role := middleware.GetRole(c)
	username := ""
	if u, exists := c.Get("username"); exists {
		if s, ok := u.(string); ok {
			username = s
		}
	}

	detailsJSON := fmt.Sprintf(`{"detail":%q}`, detail)
	h.db.Exec(`INSERT INTO audit_logs (
		tenant_id, actor_user_id, actor_role, action, target_type, target_id,
		target_user_id, request_path, request_method, outcome, details,
		created_at, user_id, username, resource, resource_id, detail, ip
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CAST(? AS jsonb), ?, ?, ?, ?, ?, ?, ?)`,
		tenantID, userID, role, action, resource, resourceID,
		"", c.Request.URL.Path, c.Request.Method, "success", detailsJSON,
		time.Now(), userID, username, resource, resourceID, detail, c.ClientIP())
}
