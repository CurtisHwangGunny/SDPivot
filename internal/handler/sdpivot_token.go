package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SDPivotTokenHandler handles token usage queries.
type SDPivotTokenHandler struct {
	db *gorm.DB
}

// NewSDPivotTokenHandler creates a new token handler.
func NewSDPivotTokenHandler(db *gorm.DB) *SDPivotTokenHandler {
	return &SDPivotTokenHandler{db: db}
}

// RegisterRoutes registers token usage routes.
func (h *SDPivotTokenHandler) RegisterRoutes(rg *gin.RouterGroup) {
	usage := rg.Group("/usage")
	{
		usage.GET("/summary", h.GetUsageSummary)
		usage.GET("/history", h.GetUsageHistory)
		usage.GET("/by-model", h.GetUsageByModel)
		usage.GET("/active-users", h.GetActiveUsers)
		usage.GET("/documents", h.GetDocumentUsage)
		usage.GET("/qa", h.GetQAUsage)
		report := usage.Group("/report", middleware.RequirePermission(middleware.PermissionDepartmentManage))
		report.GET("/by-user", h.GetUsageReportByUser)
		report.GET("/by-department", h.GetUsageReportByDepartment)
		report.GET("/export", h.ExportUsageReport)
	}
}

func (h *SDPivotTokenHandler) GetActiveUsers(c *gin.Context) {
	cutoff := time.Now().AddDate(0, 0, -30)
	var result struct {
		ActiveUsers int64 `json:"active_users"`
		Requests    int64 `json:"request_count"`
		TotalTokens int64 `json:"total_tokens"`
	}
	if err := middleware.TenantDB(c, h.db).Table("token_usage").
		Where("tenant_id = ? AND created_at >= ?", middleware.GetTenantID(c), cutoff).
		Select("COUNT(DISTINCT user_id) AS active_users, COUNT(*) AS request_count, COALESCE(SUM(input_tokens + output_tokens), 0) AS total_tokens").
		Scan(&result).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load active user usage"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"period_days": 30, "active_users": result.ActiveUsers, "request_count": result.Requests, "total_tokens": result.TotalTokens})
}

func (h *SDPivotTokenHandler) GetDocumentUsage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var result struct {
		TotalDocuments      int64 `json:"total_documents"`
		ParsedDocuments     int64 `json:"parsed_documents"`
		VectorizedDocuments int64 `json:"vectorized_documents"`
		TotalChunks         int64 `json:"total_chunks"`
		TotalBytes          int64 `json:"total_bytes"`
	}
	if err := middleware.TenantDB(c, h.db).Table("documents").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Select(`COUNT(*) AS total_documents,
			COUNT(*) FILTER (WHERE parse_status = 'completed') AS parsed_documents,
			COUNT(*) FILTER (WHERE embedding_status = 'completed') AS vectorized_documents,
			COALESCE(SUM(chunk_count), 0) AS total_chunks,
			COALESCE(SUM(file_size), 0) AS total_bytes`).Scan(&result).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load document usage"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *SDPivotTokenHandler) GetQAUsage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var qa struct {
		QuestionCount int64 `json:"question_count"`
		AnswerCount   int64 `json:"answer_count"`
		SessionCount  int64 `json:"session_count"`
	}
	if err := middleware.TenantDB(c, h.db).Table("qa_messages qm").
		Joins("JOIN qa_sessions qs ON qs.id = qm.session_id AND qs.tenant_id = qm.tenant_id").
		Where("qm.tenant_id = ?", tenantID).
		Select(`COUNT(*) FILTER (WHERE qm.role = 'user') AS question_count,
			COUNT(*) FILTER (WHERE qm.role = 'assistant') AS answer_count,
			COUNT(DISTINCT qm.session_id) AS session_count`).Scan(&qa).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load QA usage"})
		return
	}
	var tokens struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	}
	if err := middleware.TenantDB(c, h.db).Table("token_usage").Where("tenant_id = ? AND action ILIKE ?", tenantID, "%qa%").
		Select("COALESCE(SUM(input_tokens), 0) AS prompt_tokens, COALESCE(SUM(output_tokens), 0) AS completion_tokens, COALESCE(SUM(input_tokens + output_tokens), 0) AS total_tokens").
		Scan(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load QA token usage"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"question_count": qa.QuestionCount, "answer_count": qa.AnswerCount, "qa_count": qa.QuestionCount,
		"session_count": qa.SessionCount, "prompt_tokens": tokens.PromptTokens,
		"completion_tokens": tokens.CompletionTokens, "total_tokens": tokens.TotalTokens,
	})
}

// GetUsageSummary returns aggregated token usage.
func (h *SDPivotTokenHandler) GetUsageSummary(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var query types.SDPivotTokenUsageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := tenantDB.Model(&types.SDPivotTokenUsage{})
	db = db.Where("tenant_id = ?", tenantID)

	// Default: user's own usage
	if query.OrgID != nil {
		db = db.Where("org_id = ?", *query.OrgID)
	} else if query.UserID != nil {
		db = db.Where("user_id = ?", *query.UserID)
	} else {
		db = db.Where("user_id = ?", userID)
	}

	// Time range
	if query.StartAt != nil {
		if t, err := time.Parse("2006-01-02", *query.StartAt); err == nil {
			db = db.Where("created_at >= ?", t)
		}
	}
	if query.EndAt != nil {
		if t, err := time.Parse("2006-01-02", *query.EndAt); err == nil {
			db = db.Where("created_at < ?", t.Add(24*time.Hour))
		}
	}

	var summary types.SDPivotTokenUsageSummary
	db.Select(
		"COALESCE(SUM(input_tokens), 0) as total_prompt_tokens",
		"COALESCE(SUM(output_tokens), 0) as total_completion_tokens",
		"COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens",
		"COUNT(*) as request_count",
	).Scan(&summary)

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// GetUsageHistory returns token usage history grouped by time.
func (h *SDPivotTokenHandler) GetUsageHistory(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var query types.SDPivotTokenUsageQuery
	c.ShouldBindQuery(&query)

	groupBy := query.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}

	var dateFormat string
	switch groupBy {
	case "week":
		dateFormat = "YYYY-WW"
	case "month":
		dateFormat = "YYYY-MM"
	default:
		dateFormat = "YYYY-MM-DD"
	}

	db := tenantDB.Model(&types.SDPivotTokenUsage{})
	db = db.Where("tenant_id = ? AND user_id = ?", tenantID, userID)

	if query.StartAt != nil {
		if t, err := time.Parse("2006-01-02", *query.StartAt); err == nil {
			db = db.Where("created_at >= ?", t)
		}
	}

	type DailyUsage struct {
		Date         string `json:"date"`
		TotalTokens  int64  `json:"total_tokens"`
		RequestCount int64  `json:"request_count"`
	}

	var results []DailyUsage
	db.Select(
		"TO_CHAR(created_at, ?) as date, SUM(input_tokens + output_tokens) as total_tokens, COUNT(*) as request_count",
		dateFormat,
	).Group("date").Order("date").Scan(&results)

	c.JSON(http.StatusOK, gin.H{"history": results})
}

// GetUsageByModel returns token usage broken down by model.
func (h *SDPivotTokenHandler) GetUsageByModel(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	type ModelUsage struct {
		ModelID      string `json:"model_id"`
		TotalTokens  int64  `json:"total_tokens"`
		RequestCount int64  `json:"request_count"`
	}

	var results []ModelUsage
	tenantDB.Model(&types.SDPivotTokenUsage{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Select("model as model_id, SUM(input_tokens + output_tokens) as total_tokens, COUNT(*) as request_count").
		Group("model").
		Order("total_tokens DESC").
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{"models": results})
}

type sdpivotUsageReportRow struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TotalTokens      int64  `json:"total_tokens"`
	RequestCount     int64  `json:"request_count"`
}

func applySDPivotUsageRange(c *gin.Context, db *gorm.DB) *gorm.DB {
	start := strings.TrimSpace(c.Query("start_date"))
	if start == "" {
		start = strings.TrimSpace(c.Query("start_at"))
	}
	end := strings.TrimSpace(c.Query("end_date"))
	if end == "" {
		end = strings.TrimSpace(c.Query("end_at"))
	}
	if parsed, err := time.Parse("2006-01-02", start); err == nil {
		db = db.Where("tu.created_at >= ?", parsed)
	}
	if parsed, err := time.Parse("2006-01-02", end); err == nil {
		db = db.Where("tu.created_at < ?", parsed.AddDate(0, 0, 1))
	}
	return db
}

func (h *SDPivotTokenHandler) usageReportBase(c *gin.Context) *gorm.DB {
	db := middleware.TenantDB(c, h.db).Table("token_usage tu").
		Joins("LEFT JOIN users u ON u.id = tu.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN departments d ON d.id = u.department_id AND d.tenant_id = tu.tenant_id AND d.deleted_at IS NULL").
		Where("tu.tenant_id = ?", middleware.GetTenantID(c))
	if strings.EqualFold(middleware.GetRole(c), "department_admin") {
		departmentIDs, err := departmentScopeIDs(h.db, middleware.GetTenantID(c), middleware.GetDepartmentID(c))
		if err != nil || len(departmentIDs) == 0 {
			return db.Where("1 = 0")
		}
		db = db.Where("u.department_id IN ?", departmentIDs)
	}
	return applySDPivotUsageRange(c, db)
}

func (h *SDPivotTokenHandler) usageByUser(c *gin.Context) ([]sdpivotUsageReportRow, error) {
	rows := make([]sdpivotUsageReportRow, 0)
	err := h.usageReportBase(c).
		Select(`COALESCE(tu.user_id, '') AS id,
			COALESCE(NULLIF(u.username, ''), NULLIF(u.email, ''), tu.user_id, '未知用户') AS name,
			COALESCE(SUM(tu.input_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(tu.output_tokens), 0) AS completion_tokens,
			COALESCE(SUM(tu.input_tokens + tu.output_tokens), 0) AS total_tokens,
			COUNT(*) AS request_count`).
		Group("tu.user_id, u.username, u.email").Order("total_tokens DESC").Scan(&rows).Error
	return rows, err
}

func (h *SDPivotTokenHandler) usageByDepartment(c *gin.Context) ([]sdpivotUsageReportRow, error) {
	rows := make([]sdpivotUsageReportRow, 0)
	err := h.usageReportBase(c).
		Select(`COALESCE(u.department_id, '') AS id,
			COALESCE(NULLIF(d.name, ''), '未分配部门') AS name,
			COALESCE(SUM(tu.input_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(tu.output_tokens), 0) AS completion_tokens,
			COALESCE(SUM(tu.input_tokens + tu.output_tokens), 0) AS total_tokens,
			COUNT(*) AS request_count`).
		Group("u.department_id, d.name").Order("total_tokens DESC").Scan(&rows).Error
	return rows, err
}

func (h *SDPivotTokenHandler) GetUsageReportByUser(c *gin.Context) {
	rows, err := h.usageByUser(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load usage report by user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *SDPivotTokenHandler) GetUsageReportByDepartment(c *gin.Context) {
	rows, err := h.usageByDepartment(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load usage report by department"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *SDPivotTokenHandler) ExportUsageReport(c *gin.Context) {
	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv")))
	if format == "xlsx" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "xlsx export is not implemented; use format=csv"})
		return
	}
	if format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be csv or xlsx"})
		return
	}
	groupBy := strings.ToLower(strings.TrimSpace(c.DefaultQuery("group_by", "user")))
	var rows []sdpivotUsageReportRow
	var err error
	if groupBy == "department" {
		rows, err = h.usageByDepartment(c)
	} else if groupBy == "user" {
		rows, err = h.usageByUser(c)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group_by must be user or department"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export usage report"})
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=usage_report_"+groupBy+".csv")
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"ID", "Name", "Prompt Tokens", "Completion Tokens", "Total Tokens", "Request Count"})
	for _, row := range rows {
		_ = writer.Write([]string{row.ID, row.Name, strconv.FormatInt(row.PromptTokens, 10), strconv.FormatInt(row.CompletionTokens, 10), strconv.FormatInt(row.TotalTokens, 10), strconv.FormatInt(row.RequestCount, 10)})
	}
	writer.Flush()
}
