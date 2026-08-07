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

var sdpivotSecuritySections = map[string]struct{}{
	"ip_whitelist": {}, "password_policy": {}, "session": {}, "desensitize": {}, "audit": {},
}

type sdpivotSecuritySetting struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64 `gorm:"uniqueIndex:idx_sdp_security_setting"`
	Section   string `gorm:"uniqueIndex:idx_sdp_security_setting"`
	Key       string `gorm:"uniqueIndex:idx_sdp_security_setting"`
	Value     string
	UpdatedAt time.Time
}

func (sdpivotSecuritySetting) TableName() string { return "security_settings" }

type SDPivotSecurityHandler struct{ db *gorm.DB }

func NewSDPivotSecurityHandler(db *gorm.DB) *SDPivotSecurityHandler {
	return &SDPivotSecurityHandler{db: db}
}

func (h *SDPivotSecurityHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin/security", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	admin.GET("", h.Get)
	admin.PUT("/:section", h.UpdateSection)
}

func (h *SDPivotSecurityHandler) Get(c *gin.Context) {
	rows := make([]sdpivotSecuritySetting, 0)
	if err := middleware.TenantDB(c, h.db).Where("tenant_id = ?", middleware.GetTenantID(c)).Order("section, key").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load security settings"})
		return
	}
	settings := make(map[string]map[string]string, len(sdpivotSecuritySections))
	for section := range sdpivotSecuritySections {
		settings[section] = map[string]string{}
	}
	for _, row := range rows {
		settings[row.Section][row.Key] = row.Value
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

func (h *SDPivotSecurityHandler) UpdateSection(c *gin.Context) {
	section := strings.TrimSpace(c.Param("section"))
	if _, ok := sdpivotSecuritySections[section]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid security section"})
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
		row := sdpivotSecuritySetting{ID: uuid.NewString(), TenantID: middleware.GetTenantID(c), Section: section, Key: key, Value: value, UpdatedAt: now}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "section"}, {Name: "key"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"value": value, "updated_at": now}),
		}).Create(&row).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save security settings"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "security settings saved", "section": section})
}
