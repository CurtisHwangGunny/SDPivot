package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
)

const sdpivotToolingVersion = "2.0.0"

type sdpivotMCPService struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64    `json:"tenant_id"`
	Name      string    `json:"name"`
	Transport string    `json:"transport"`
	Endpoint  string    `json:"endpoint"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (sdpivotMCPService) TableName() string { return "mcp_services" }

// SDPivotToolingHandler exposes discovery endpoints for SDPivot CLI and MCP clients.
type SDPivotToolingHandler struct {
	db *gorm.DB
}

// NewSDPivotToolingHandler creates a tooling discovery handler.
func NewSDPivotToolingHandler(db *gorm.DB) *SDPivotToolingHandler {
	return &SDPivotToolingHandler{db: db}
}

// RegisterRoutes registers public CLI discovery and protected MCP configuration routes.
func (h *SDPivotToolingHandler) RegisterRoutes(public, protected *gin.RouterGroup) {
	public.GET("/sdpivot/cli/info", h.CLIInfo)
	mcp := protected.Group("/sdpivot/mcp", middleware.RequirePermission(middleware.PermissionKnowledgeRead))
	mcp.GET("/config", h.MCPConfig)
}

// CLIInfo returns the CLI version and concise command usage.
func (h *SDPivotToolingHandler) CLIInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"brand":   "SDPivot·文枢",
		"version": sdpivotToolingVersion,
		"usage": []string{
			"sdpivot-cli [--server URL] auth --email EMAIL --password PASSWORD",
			"sdpivot-cli [--server URL] search --query QUERY",
			"sdpivot-cli [--server URL] doc list --space SPACE_ID",
			"sdpivot-cli [--server URL] doc upload --space SPACE_ID --file PATH",
			"sdpivot-cli [--server URL] qa --session SESSION_ID --content CONTENT",
		},
	})
}

// MCPConfig lists enabled MCP registrations for the current tenant.
func (h *SDPivotToolingHandler) MCPConfig(c *gin.Context) {
	services := make([]sdpivotMCPService, 0)
	err := middleware.TenantDB(c, h.db).
		Where("tenant_id = ? AND enabled = ?", middleware.GetTenantID(c), true).
		Order("created_at ASC").
		Find(&services).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load MCP configuration"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"services": services})
}
