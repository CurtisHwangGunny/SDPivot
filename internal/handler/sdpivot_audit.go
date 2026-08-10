package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

const (
	auditModuleUser     = "user"
	auditModuleDocument = "document"
	auditModuleSpace    = "space"
)

const (
	auditActionLoginSuccess    types.AuditAction = "auth.login_success"
	auditActionUserCreated     types.AuditAction = "user.created"
	auditActionUserImported    types.AuditAction = "user.imported"
	auditActionUserRoleUpdated types.AuditAction = "user.role_updated"
	auditActionUserStatus      types.AuditAction = "user.status_updated"
	auditActionDocumentUpload  types.AuditAction = "document.uploaded"
	auditActionDocumentDelete  types.AuditAction = "document.deleted"
	auditActionSpaceCreate     types.AuditAction = "space.created"
	auditActionSpaceDelete     types.AuditAction = "space.deleted"
)

// SDPivotAuditHandler exposes tenant-scoped audit and dashboard endpoints.
type SDPivotAuditHandler struct {
	db *gorm.DB
}

// NewSDPivotAuditHandler creates a tenant-scoped audit handler.
func NewSDPivotAuditHandler(db *gorm.DB) *SDPivotAuditHandler {
	return &SDPivotAuditHandler{db: db}
}

// RegisterRoutes registers SDPivot admin audit and dashboard routes.
func (h *SDPivotAuditHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	{
		admin.GET("/audit", h.ListAuditLogs)
		admin.GET("/audit/login", h.ListLoginLogs)
		admin.GET("/audit/knowledge", h.ListKnowledgeLogs)
		admin.GET("/audit/export", h.ExportAuditLogs)
		admin.GET("/dashboard", h.GetDashboard)
	}
	auditToplevel := rg.Group("/audit", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	{
		auditToplevel.GET("/logs", h.ListAuditLogs)
		auditToplevel.GET("", h.ListAuditLogs)
		auditToplevel.GET("/login", h.ListLoginLogs)
		auditToplevel.GET("/knowledge", h.ListKnowledgeLogs)
		auditToplevel.GET("/export", h.ExportAuditLogs)
	}
}

type sdpivotAuditRow struct {
	ID            uint64             `json:"id"`
	TenantID      uint64             `json:"tenant_id"`
	UserID        string             `json:"user_id"`
	Username      string             `json:"username"`
	ActorRole     string             `json:"actor_role"`
	Module        string             `json:"module"`
	Action        types.AuditAction  `json:"action"`
	ResourceType  string             `json:"resource_type"`
	ResourceID    string             `json:"resource_id"`
	RequestPath   string             `json:"request_path"`
	RequestMethod string             `json:"request_method"`
	Outcome       types.AuditOutcome `json:"outcome"`
	Details       types.JSON         `json:"details"`
	IPAddress     string             `json:"ip_address"`
	CreatedAt     time.Time          `json:"created_at"`
}

func auditModuleExpression(db *gorm.DB) string {
	if db.Dialector.Name() == "sqlite" {
		return "CASE WHEN al.action LIKE 'auth.login%' THEN 'login' WHEN COALESCE(NULLIF(al.resource_type, ''), al.target_type) IN ('space', 'document') THEN 'knowledge' WHEN instr(al.action, '.') > 0 THEN substr(al.action, 1, instr(al.action, '.') - 1) ELSE al.action END"
	}
	return "CASE WHEN al.action LIKE 'auth.login%' THEN 'login' WHEN COALESCE(NULLIF(al.resource_type, ''), al.target_type) IN ('space', 'document') THEN 'knowledge' ELSE split_part(al.action, '.', 1) END"
}

func (h *SDPivotAuditHandler) auditQuery(c *gin.Context, fixedModule string) *gorm.DB {
	tenantID := middleware.GetTenantID(c)
	db := middleware.TenantDB(c, h.db)
	moduleExpression := auditModuleExpression(db)
	query := db.Table("audit_logs al").
		Select(fmt.Sprintf(`al.id, al.tenant_id,
			COALESCE(NULLIF(al.user_id, ''), al.actor_user_id) AS user_id,
			al.username AS username,
			al.actor_role, %s AS module, al.action,
			COALESCE(NULLIF(al.resource_type, ''), al.target_type) AS resource_type,
			COALESCE(NULLIF(al.resource_id, ''), al.target_id) AS resource_id,
			al.request_path, al.request_method, al.outcome, al.details,
			al.ip_address, al.created_at`, moduleExpression)).
		Where("al.tenant_id = ?", tenantID)

	module := strings.TrimSpace(fixedModule)
	if module == "" {
		module = strings.TrimSpace(c.Query("module"))
	}
	if module != "" {
		query = query.Where(moduleExpression+" = ?", module)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		query = query.Where("al.action = ?", action)
	}
	return query
}

func (h *SDPivotAuditHandler) listAuditLogs(c *gin.Context, module string) {
	page, pageSize := parsePagination(c)
	query := h.auditQuery(c, module)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count audit logs"})
		return
	}

	rows := make([]sdpivotAuditRow, 0)
	if err := query.Order("al.created_at DESC, al.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": rows, "total": total, "page": page, "page_size": pageSize})
}

// ListAuditLogs returns filtered administrator operations.
func (h *SDPivotAuditHandler) ListAuditLogs(c *gin.Context) {
	h.listAuditLogs(c, "")
}

// ListLoginLogs returns successful login records.
func (h *SDPivotAuditHandler) ListLoginLogs(c *gin.Context) {
	page, pageSize := parsePagination(c)
	query := h.auditQuery(c, "").Where("al.action LIKE ?", "auth.login%")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count login audit logs"})
		return
	}
	rows := make([]sdpivotAuditRow, 0)
	if err := query.Order("al.created_at DESC, al.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list login audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": rows, "total": total, "page": page, "page_size": pageSize})
}

// ListKnowledgeLogs returns knowledge space/document activity.
func (h *SDPivotAuditHandler) ListKnowledgeLogs(c *gin.Context) {
	page, pageSize := parsePagination(c)
	query := h.auditQuery(c, "").Where("COALESCE(NULLIF(al.resource_type, ''), al.target_type) IN ?", []string{"space", "document"})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count knowledge audit logs"})
		return
	}
	rows := make([]sdpivotAuditRow, 0)
	if err := query.Order("al.created_at DESC, al.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list knowledge audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": rows, "total": total, "page": page, "page_size": pageSize})
}

// ExportAuditLogs exports the current tenant's filtered logs as CSV.
func (h *SDPivotAuditHandler) ExportAuditLogs(c *gin.Context) {
	rows := make([]sdpivotAuditRow, 0)
	if err := h.auditQuery(c, "").Order("al.created_at DESC, al.id DESC").Limit(10000).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export audit logs"})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=sdpivot_audit_logs.csv")
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"ID", "Time", "Module", "Action", "User", "Role", "Resource Type", "Resource ID", "Outcome", "IP", "Details"})
	for _, row := range rows {
		_ = writer.Write([]string{
			strconv.FormatUint(row.ID, 10), row.CreatedAt.Format(time.RFC3339), row.Module, string(row.Action),
			row.Username, row.ActorRole, row.ResourceType, row.ResourceID, string(row.Outcome), row.IPAddress, string(row.Details),
		})
	}
	writer.Flush()
}

type sdpivotTrendPoint struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

type sdpivotDashboardResponse struct {
	QATrend       []sdpivotTrendPoint `json:"qa_trend"`
	DocumentTrend []sdpivotTrendPoint `json:"document_trend"`
	TokenTrend    []sdpivotTrendPoint `json:"token_trend"`
}

func trendDateExpression(db *gorm.DB, column string) string {
	if db.Dialector.Name() == "sqlite" {
		return "strftime('%Y-%m-%d', " + column + ")"
	}
	return "TO_CHAR(" + column + ", 'YYYY-MM-DD')"
}

func sevenDayTrend(now time.Time, values map[string]int64) []sdpivotTrendPoint {
	trend := make([]sdpivotTrendPoint, 0, 7)
	for daysAgo := 6; daysAgo >= 0; daysAgo-- {
		date := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
		trend = append(trend, sdpivotTrendPoint{Date: date, Value: values[date]})
	}
	return trend
}

func scanTrend(query *gorm.DB) (map[string]int64, error) {
	rows := make([]sdpivotTrendPoint, 0)
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	values := make(map[string]int64, len(rows))
	for _, row := range rows {
		values[row.Date] = row.Value
	}
	return values, nil
}

// GetDashboard returns seven-day QA, document, and token trends.
func (h *SDPivotAuditHandler) GetDashboard(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)

	qaDate := trendDateExpression(db, "created_at")
	qaValues, err := scanTrend(db.Table("qa_messages").Select(qaDate+" AS date, COUNT(*) AS value").
		Where("tenant_id = ? AND role = ? AND created_at >= ?", tenantID, "user", start).Group(qaDate).Order(qaDate))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load qa trend"})
		return
	}

	documentDate := trendDateExpression(db, "created_at")
	documentValues, err := scanTrend(db.Table("documents").Select(documentDate+" AS date, COUNT(*) AS value").
		Where("tenant_id = ? AND deleted_at IS NULL AND created_at >= ?", tenantID, start).Group(documentDate).Order(documentDate))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load document trend"})
		return
	}

	tokenDate := trendDateExpression(db, "created_at")
	tokenValues, err := scanTrend(db.Table("token_usage").Select(tokenDate+" AS date, COALESCE(SUM(input_tokens + output_tokens), 0) AS value").
		Where("tenant_id = ? AND created_at >= ?", tenantID, start).Group(tokenDate).Order(tokenDate))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load token trend"})
		return
	}

	c.JSON(http.StatusOK, sdpivotDashboardResponse{
		QATrend: sevenDayTrend(now, qaValues), DocumentTrend: sevenDayTrend(now, documentValues), TokenTrend: sevenDayTrend(now, tokenValues),
	})
}

func writeSDPivotAuditLog(db *gorm.DB, c *gin.Context, action types.AuditAction, module, resourceType, resourceID string, details map[string]interface{}) {
	if db == nil || c == nil {
		return
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte("{}")
	}
	userID := middleware.GetUserID(c)
	entry := types.AuditLog{
		TenantID: middleware.GetTenantID(c), ActorUserID: userID, UserID: userID, ActorRole: middleware.GetRole(c),
		Action: action, TargetType: resourceType, TargetID: resourceID, ResourceType: resourceType, ResourceID: resourceID,
		RequestPath: c.FullPath(), RequestMethod: c.Request.Method, Outcome: types.AuditOutcomeSuccess,
		Details: types.JSON(detailsJSON), IPAddress: c.ClientIP(), CreatedAt: time.Now(),
	}
	if entry.TenantID == 0 {
		entry.TenantID = types.DefaultTenantID
	}
	if module != "" && !strings.HasPrefix(string(action), module+".") {
		entry.Details = types.JSON(detailsJSON)
	}
	_ = db.Create(&entry).Error
}

func writeSDPivotLoginAuditLog(db *gorm.DB, c *gin.Context, user *types.User) {
	if db == nil || c == nil || user == nil {
		return
	}
	role := resolveUserRole(*user)
	details, _ := json.Marshal(map[string]interface{}{"login_type": "password"})
	entry := types.AuditLog{
		TenantID: types.DefaultTenantID, ActorUserID: user.ID, UserID: user.ID, ActorRole: role,
		Action: auditActionLoginSuccess, TargetType: "user", TargetID: user.ID, ResourceType: "user", ResourceID: user.ID,
		RequestPath: c.FullPath(), RequestMethod: c.Request.Method, Outcome: types.AuditOutcomeSuccess,
		Details: types.JSON(details), IPAddress: c.ClientIP(), CreatedAt: time.Now(),
	}
	_ = db.Create(&entry).Error
}
