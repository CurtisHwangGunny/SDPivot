package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/middleware"
)

var sdpivotConfigSections = map[string]struct{}{
	"storage": {}, "sms": {}, "wechat": {}, "search": {}, "cli_mcp": {}, "global": {},
}

type sdpivotSystemSetting struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64 `gorm:"uniqueIndex:idx_sdp_system_setting"`
	Section   string `gorm:"uniqueIndex:idx_sdp_system_setting"`
	Key       string `gorm:"uniqueIndex:idx_sdp_system_setting"`
	Value     string
	UpdatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (sdpivotSystemSetting) TableName() string { return "system_settings" }

type SDPivotConfigHandler struct{ db *gorm.DB }

func NewSDPivotConfigHandler(db *gorm.DB) *SDPivotConfigHandler { return &SDPivotConfigHandler{db: db} }

func (h *SDPivotConfigHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin/config", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	admin.GET("", h.Get)
	admin.PUT("/:section", h.UpdateSection)
}

func (h *SDPivotConfigHandler) Get(c *gin.Context) {
	rows := make([]sdpivotSystemSetting, 0)
	if err := middleware.TenantDB(c, h.db).Where("tenant_id = ?", middleware.GetTenantID(c)).Order("section, key").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load system settings"})
		return
	}
	settings := make(map[string]map[string]string, len(sdpivotConfigSections))
	for section := range sdpivotConfigSections {
		settings[section] = map[string]string{}
	}
	for _, row := range rows {
		settings[row.Section][row.Key] = row.Value
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

func (h *SDPivotConfigHandler) UpdateSection(c *gin.Context) {
	section := strings.TrimSpace(c.Param("section"))
	if _, ok := sdpivotConfigSections[section]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config section"})
		return
	}
	var values map[string]string
	if err := c.ShouldBindJSON(&values); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "settings must be a string map"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	now := time.Now()
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || len(key) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "setting keys must be 1-100 characters"})
			return
		}
		row := sdpivotSystemSetting{ID: uuid.NewString(), TenantID: middleware.GetTenantID(c), Section: section, Key: key, Value: value, UpdatedBy: middleware.GetUserID(c), CreatedAt: now, UpdatedAt: now}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "section"}, {Name: "key"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"value": value, "updated_by": middleware.GetUserID(c), "updated_at": now}),
		}).Create(&row).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save system settings"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "system settings saved", "section": section})
}
