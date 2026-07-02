package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SmartKnoraTokenHandler handles token usage queries.
type SmartKnoraTokenHandler struct {
	db *gorm.DB
}

// NewSmartKnoraTokenHandler creates a new token handler.
func NewSmartKnoraTokenHandler(db *gorm.DB) *SmartKnoraTokenHandler {
	return &SmartKnoraTokenHandler{db: db}
}

// RegisterRoutes registers token usage routes.
func (h *SmartKnoraTokenHandler) RegisterRoutes(rg *gin.RouterGroup) {
	usage := rg.Group("/usage")
	{
		usage.GET("/summary", h.GetUsageSummary)
		usage.GET("/history", h.GetUsageHistory)
		usage.GET("/by-model", h.GetUsageByModel)
	}
}

// GetUsageSummary returns aggregated token usage.
func (h *SmartKnoraTokenHandler) GetUsageSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var query types.SmartKnoraTokenUsageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := h.db.Model(&types.SmartKnoraTokenUsage{})
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

	var summary types.SmartKnoraTokenUsageSummary
	db.Select(
		"COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens",
		"COALESCE(SUM(completion_tokens), 0) as total_completion_tokens",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COUNT(*) as request_count",
	).Scan(&summary)

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// GetUsageHistory returns token usage history grouped by time.
func (h *SmartKnoraTokenHandler) GetUsageHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var query types.SmartKnoraTokenUsageQuery
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

	db := h.db.Model(&types.SmartKnoraTokenUsage{})
	db = db.Where("tenant_id = ? AND user_id = ?", tenantID, userID)

	if query.StartAt != nil {
		if t, err := time.Parse("2006-01-02", *query.StartAt); err == nil {
			db = db.Where("created_at >= ?", t)
		}
	}

	type DailyUsage struct {
		Date          string `json:"date"`
		TotalTokens   int64  `json:"total_tokens"`
		RequestCount  int64  `json:"request_count"`
	}

	var results []DailyUsage
	db.Select(
		"TO_CHAR(created_at, ?) as date, SUM(total_tokens) as total_tokens, COUNT(*) as request_count",
		dateFormat,
	).Group("date").Order("date").Scan(&results)

	c.JSON(http.StatusOK, gin.H{"history": results})
}

// GetUsageByModel returns token usage broken down by model.
func (h *SmartKnoraTokenHandler) GetUsageByModel(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	type ModelUsage struct {
		ModelID      string `json:"model_id"`
		TotalTokens  int64  `json:"total_tokens"`
		RequestCount int64  `json:"request_count"`
	}

	var results []ModelUsage
	h.db.Model(&types.SmartKnoraTokenUsage{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Select("model_id, SUM(total_tokens) as total_tokens, COUNT(*) as request_count").
		Group("model_id").
		Order("total_tokens DESC").
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{"models": results})
}
