package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SDPivotAdminHandler exposes SDPivot-only administration compatibility APIs.
type SDPivotAdminHandler struct {
	db *gorm.DB
}

func NewSDPivotAdminHandler(db *gorm.DB) *SDPivotAdminHandler {
	return &SDPivotAdminHandler{db: db}
}

func (h *SDPivotAdminHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Read-only admin routes: accessible to all authenticated users (viewer can read)
	adminRead := rg.Group("/admin")
	adminRead.GET("/tags", h.ListTagDictionary)

	// Write routes: require elevated permission
	adminWrite := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionUserRoleAssign))
	adminWrite.POST("/tags", h.CreateTagDictionaryEntry)
	adminWrite.PUT("/tags/:id", h.UpdateTagDictionaryEntry)
	adminWrite.DELETE("/tags/:id", h.DeleteTagDictionaryEntry)

	adminTags := rg.Group("/admin/tags", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	adminTags.GET("/dictionary", h.GetRecommendedTagDictionary)
	adminTags.GET("/adjustment-log", h.ListTagAdjustmentLog)
	adminTags.GET("/learning", h.GetTagLearningStatus)
}

func (h *SDPivotAdminHandler) GetRecommendedTagDictionary(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	type dictionaryRow struct {
		DimensionID   string `json:"dimension_id"`
		DimensionCode string `json:"dimension_code"`
		DimensionName string `json:"dimension_name"`
		TagID         string `json:"tag_id"`
		TagName       string `json:"tag_name"`
		Color         string `json:"color"`
	}
	rows := make([]dictionaryRow, 0)
	if err := db.Table("tag_dimensions dim").
		Joins("LEFT JOIN tag_dictionary dict ON dict.dimension_id = dim.id").
		Where("dim.deleted_at IS NULL AND dim.enabled = TRUE AND (dim.tenant_id IS NULL OR dim.tenant_id = ?)", tenantID).
		Select("dim.id AS dimension_id, dim.code AS dimension_code, dim.name AS dimension_name, COALESCE(dict.id, '') AS tag_id, COALESCE(dict.name, '') AS tag_name, COALESCE(dict.color, '') AS color").
		Order("dim.sort_order, dict.sort_order, dict.name").Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tag dictionary"})
		return
	}
	type dimension struct {
		ID     string  `json:"dimension_id"`
		Code   string  `json:"code"`
		Name   string  `json:"name"`
		Values []gin.H `json:"values"`
	}
	items := make([]dimension, 0)
	indexes := make(map[string]int)
	for _, row := range rows {
		index, exists := indexes[row.DimensionID]
		if !exists {
			index = len(items)
			indexes[row.DimensionID] = index
			items = append(items, dimension{ID: row.DimensionID, Code: row.DimensionCode, Name: row.DimensionName, Values: make([]gin.H, 0)})
		}
		if row.TagID != "" {
			items[index].Values = append(items[index].Values, gin.H{"id": row.TagID, "name": row.TagName, "color": row.Color})
		}
	}
	c.JSON(http.StatusOK, gin.H{"dimensions": items})
}

func (h *SDPivotAdminHandler) ListTagAdjustmentLog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	db := middleware.TenantDB(c, h.db).Table("tag_adjustment_log log").
		Joins("LEFT JOIN users operator ON operator.id::text = log.operator::text AND operator.tenant_id = log.tenant_id").
		Joins("LEFT JOIN documents doc ON doc.id = log.document_id AND doc.tenant_id = log.tenant_id").
		Joins("LEFT JOIN tag_dictionary old_tag ON old_tag.id = log.old_tag_id").
		Joins("LEFT JOIN tag_dictionary new_tag ON new_tag.id = log.new_tag_id").
		Where("log.tenant_id = ?", middleware.GetTenantID(c))
	var total int64
	if err := db.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count tag adjustments"})
		return
	}
	type adjustment struct {
		ID            string    `json:"id"`
		CreatedAt     time.Time `json:"time"`
		OperatorID    string    `json:"operator_id"`
		OperatorName  string    `json:"operator_name"`
		DocumentID    string    `json:"document_id"`
		DocumentTitle string    `json:"document_title"`
		OldTagID      *string   `json:"old_tag_id"`
		OldTagName    string    `json:"old_tag_name"`
		NewTagID      *string   `json:"new_tag_id"`
		NewTagName    string    `json:"new_tag_name"`
		Reason        string    `json:"reason"`
		Status        string    `json:"status"`
	}
	items := make([]adjustment, 0)
	if err := db.Select(`log.id, log.created_at, log.operator AS operator_id,
		COALESCE(NULLIF(operator.username, ''), NULLIF(operator.email, ''), log.operator) AS operator_name,
		log.document_id, COALESCE(doc.title, '') AS document_title, log.old_tag_id,
		COALESCE(old_tag.name, '') AS old_tag_name, log.new_tag_id,
		COALESCE(new_tag.name, '') AS new_tag_name, log.reason, log.status`).
		Order("log.created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tag adjustments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *SDPivotAdminHandler) GetTagLearningStatus(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))
	if days < 1 || days > 90 {
		days = 14
	}
	cutoff := time.Now().AddDate(0, 0, -(days - 1)).Truncate(24 * time.Hour)
	type sampleSummary struct {
		SampleCount int64
		Correct     int64
	}
	var summary sampleSummary
	db := middleware.TenantDB(c, h.db)
	if err := db.Table("tag_feedback_queue").Where("tenant_id = ?", middleware.GetTenantID(c)).
		Select("COUNT(*) AS sample_count, COUNT(*) FILTER (WHERE feedback = 'correct') AS correct").Scan(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tag learning status"})
		return
	}
	type trendRow struct {
		Date    time.Time
		Samples int64
		Correct int64
	}
	var rows []trendRow
	if err := db.Table("tag_feedback_queue").Where("tenant_id = ? AND created_at >= ?", middleware.GetTenantID(c), cutoff).
		Select("DATE_TRUNC('day', created_at) AS date, COUNT(*) AS samples, COUNT(*) FILTER (WHERE feedback = 'correct') AS correct").
		Group("DATE_TRUNC('day', created_at)").Order("date").Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tag learning trend"})
		return
	}
	byDate := make(map[string]trendRow, len(rows))
	for _, row := range rows {
		byDate[row.Date.Format("2006-01-02")] = row
	}
	trend := make([]gin.H, 0, days)
	for index := 0; index < days; index++ {
		date := cutoff.AddDate(0, 0, index).Format("2006-01-02")
		row := byDate[date]
		accuracy := float64(0)
		if row.Samples > 0 {
			accuracy = float64(row.Correct) / float64(row.Samples)
		}
		trend = append(trend, gin.H{"date": date, "sample_count": row.Samples, "accuracy": accuracy})
	}
	accuracy := float64(0)
	if summary.SampleCount > 0 {
		accuracy = float64(summary.Correct) / float64(summary.SampleCount)
	}
	c.JSON(http.StatusOK, gin.H{"enabled": true, "sample_count": summary.SampleCount, "recent_accuracy": accuracy, "trend": trend})
}

// ListTagDictionary returns the shared platform classification dictionary.
func (h *SDPivotAdminHandler) ListTagDictionary(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	if tenantID == 0 {
		dimensions, tags, err := repository.NewDocumentTagRepository(db).ListClassificationDictionary(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tag dictionary"})
			return
		}
		if dimensions == nil {
			dimensions = make([]*types.TagDimension, 0)
		}
		if tags == nil {
			tags = make([]*types.TagDictionary, 0)
		}
		c.JSON(http.StatusOK, gin.H{"dimensions": dimensions, "tags": tags})
		return
	}
	var dimensions []*types.SDPivotTagDimension
	if err := db.WithContext(c.Request.Context()).
		Where("deleted_at IS NULL AND (tenant_id IS NULL OR tenant_id = ?)", tenantID).
		Order("sort_order ASC, code ASC").Find(&dimensions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tag dictionary"})
		return
	}
	dimensionIDs := make([]string, 0, len(dimensions))
	for _, dimension := range dimensions {
		dimensionIDs = append(dimensionIDs, dimension.ID)
	}
	var tags []*types.TagDictionary
	if len(dimensionIDs) > 0 {
		if err := db.WithContext(c.Request.Context()).Where("dimension_id IN ?", dimensionIDs).
			Order("dimension_id ASC, sort_order ASC, name ASC").Find(&tags).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tag dictionary"})
			return
		}
	}
	if dimensions == nil {
		dimensions = make([]*types.SDPivotTagDimension, 0)
	}
	if tags == nil {
		tags = make([]*types.TagDictionary, 0)
	}
	c.JSON(http.StatusOK, gin.H{"dimensions": dimensions, "tags": tags})
}

func (h *SDPivotAdminHandler) tagSystemHandler(c *gin.Context) *SystemHandler {
	return &SystemHandler{db: middleware.TenantDB(c, h.db)}
}

func (h *SDPivotAdminHandler) CreateTagDictionaryEntry(c *gin.Context) {
	h.tagSystemHandler(c).CreateTagDictionaryEntry(c)
}

func (h *SDPivotAdminHandler) UpdateTagDictionaryEntry(c *gin.Context) {
	h.tagSystemHandler(c).UpdateTagDictionaryEntry(c)
}

func (h *SDPivotAdminHandler) DeleteTagDictionaryEntry(c *gin.Context) {
	h.tagSystemHandler(c).DeleteTagDictionaryEntry(c)
}
