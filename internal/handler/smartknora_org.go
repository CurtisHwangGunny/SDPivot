package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SmartKnoraOrgHandler handles enterprise/organization management.
type SmartKnoraOrgHandler struct {
	db *gorm.DB
}

// NewSmartKnoraOrgHandler creates a new org handler.
func NewSmartKnoraOrgHandler(db *gorm.DB) *SmartKnoraOrgHandler {
	return &SmartKnoraOrgHandler{db: db}
}

// RegisterRoutes registers org management routes.
func (h *SmartKnoraOrgHandler) RegisterRoutes(rg *gin.RouterGroup) {
	orgs := rg.Group("/organizations")
	{
		orgs.POST("", h.CreateOrganization)
		orgs.GET("", h.ListOrganizations)
		orgs.GET("/:id", h.GetOrganization)
		orgs.PUT("/:id", h.UpdateOrganization)
		orgs.POST("/join", h.JoinOrganization)
		orgs.GET("/:id/members", h.ListMembers)
		orgs.POST("/:id/members", h.AddMember)
		orgs.DELETE("/:id/members/:userId", h.RemoveMember)
		orgs.PUT("/:id/members/:userId/role", h.UpdateMemberRole)
	}
}

// CreateOrganization creates a new enterprise organization.
func (h *SmartKnoraOrgHandler) CreateOrganization(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	var req types.CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	now := time.Now()

	// Create WeKnora Organization
	org := types.Organization{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.LogoURL,
		OwnerID:     userID,
		InviteCode:  generateInviteCode(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := tenantDB.Create(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	// Create smartKnora extension with auth status
	authExpires := now.Add(30 * 24 * time.Hour) // 30-day trial
	orgExt := types.OrgExt{
		OrgID:         org.ID,
		AuthStatus:    "trial",
		AuthType:      "trial_30d",
		AuthExpiresAt: &authExpires,
		TenantID:      tenantID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	tenantDB.Create(&orgExt)

	// Add owner as org member
	member := types.SmartKnoraOrgMember{
		ID:       uuid.New().String(),
		OrgID:    org.ID,
		UserID:   userID,
		Role:     "owner",
		Status:   "active",
		JoinedAt: now,
	}
	tenantDB.Create(&member)

	c.JSON(http.StatusCreated, gin.H{
		"organization": org,
		"auth_status":  orgExt.AuthStatus,
		"auth_expires": orgExt.AuthExpiresAt,
	})
}

// ListOrganizations lists organizations the current user belongs to.
func (h *SmartKnoraOrgHandler) ListOrganizations(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)

	var members []types.SmartKnoraOrgMember
	tenantDB.Where("user_id = ? AND status = 'active'", userID).Find(&members)

	orgIDs := make([]string, len(members))
	for i, m := range members {
		orgIDs[i] = m.OrgID
	}

	var orgs []types.Organization
	if len(orgIDs) > 0 {
		tenantDB.Where("id IN ?", orgIDs).Find(&orgs)
	}

	// Get auth status for each org
	orgExts := make(map[string]types.OrgExt)
	var exts []types.OrgExt
	tenantDB.Where("org_id IN ?", orgIDs).Find(&exts)
	for _, ext := range exts {
		orgExts[ext.OrgID] = ext
	}

	result := make([]gin.H, len(orgs))
	for i, org := range orgs {
		ext := orgExts[org.ID]
		result[i] = gin.H{
			"organization":   org,
			"auth_status":    ext.AuthStatus,
			"days_remaining": ext.DaysRemaining(),
		}
	}

	c.JSON(http.StatusOK, gin.H{"organizations": result})
}

// GetOrganization gets a specific organization.
func (h *SmartKnoraOrgHandler) GetOrganization(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	orgID := c.Param("id")

	var org types.Organization
	if err := tenantDB.Where("id = ?", orgID).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	var ext types.OrgExt
	tenantDB.Where("org_id = ?", orgID).First(&ext)

	c.JSON(http.StatusOK, gin.H{
		"organization":   org,
		"auth_status":    ext.AuthStatus,
		"auth_expires":   ext.AuthExpiresAt,
		"days_remaining": ext.DaysRemaining(),
	})
}

// UpdateOrganization updates org info.
func (h *SmartKnoraOrgHandler) UpdateOrganization(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	orgID := c.Param("id")
	userID := middleware.GetUserID(c)

	// Verify ownership
	var org types.Organization
	if err := tenantDB.Where("id = ? AND owner_id = ?", orgID, userID).First(&org).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		LogoURL     *string `json:"logo_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.LogoURL != nil {
		updates["avatar"] = *req.LogoURL
	}

	tenantDB.Model(&org).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"organization": org})
}

// JoinOrganization allows a user to join via invite code.
func (h *SmartKnoraOrgHandler) JoinOrganization(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	var req types.JoinOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)

	// Find org by invite code
	var org types.Organization
	if err := tenantDB.Where("invite_code = ?", req.InviteCode).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid invite code"})
		return
	}

	// Check if already a member
	var existing types.SmartKnoraOrgMember
	if err := tenantDB.Where("org_id = ? AND user_id = ?", org.ID, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already a member"})
		return
	}

	// Add as member
	member := types.SmartKnoraOrgMember{
		ID:       uuid.New().String(),
		OrgID:    org.ID,
		UserID:   userID,
		Role:     "viewer",
		Status:   "active",
		JoinedAt: time.Now(),
	}
	tenantDB.Create(&member)

	c.JSON(http.StatusOK, gin.H{"message": "joined successfully", "organization": org})
}

// ListMembers lists members of an organization.
func (h *SmartKnoraOrgHandler) ListMembers(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	orgID := c.Param("id")

	var members []types.SmartKnoraOrgMember
	tenantDB.Where("org_id = ? AND status = 'active'", orgID).Find(&members)

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddMember adds a member to an organization.
func (h *SmartKnoraOrgHandler) AddMember(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	orgID := c.Param("id")
	callerID := middleware.GetUserID(c)

	// Verify caller is owner/admin of the org
	var caller types.SmartKnoraOrgMember
	if err := tenantDB.Where("org_id = ? AND user_id = ? AND role IN ('owner','admin')", orgID, callerID).First(&caller).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to add members"})
		return
	}

	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"omitempty,oneof=admin editor viewer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role == "" {
		req.Role = "viewer"
	}

	member := types.SmartKnoraOrgMember{
		ID:       uuid.New().String(),
		OrgID:    orgID,
		UserID:   req.UserID,
		Role:     req.Role,
		Status:   "active",
		JoinedAt: time.Now(),
	}
	tenantDB.Create(&member)

	c.JSON(http.StatusCreated, gin.H{"member": member})
}

// RemoveMember removes a member from an organization.
func (h *SmartKnoraOrgHandler) RemoveMember(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	orgID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID := middleware.GetUserID(c)

	// Verify caller is owner/admin of the org
	var caller types.SmartKnoraOrgMember
	if err := tenantDB.Where("org_id = ? AND user_id = ? AND role IN ('owner','admin')", orgID, callerID).First(&caller).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to remove members"})
		return
	}

	if err := tenantDB.Where("org_id = ? AND user_id = ?", orgID, targetUserID).Delete(&types.SmartKnoraOrgMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}

// UpdateMemberRole updates a member's role.
func (h *SmartKnoraOrgHandler) UpdateMemberRole(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	orgID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID := middleware.GetUserID(c)

	// Verify caller is owner/admin of the org
	var caller types.SmartKnoraOrgMember
	if err := tenantDB.Where("org_id = ? AND user_id = ? AND role IN ('owner','admin')", orgID, callerID).First(&caller).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to update member roles"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin editor viewer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantDB.Model(&types.SmartKnoraOrgMember{}).
		Where("org_id = ? AND user_id = ?", orgID, targetUserID).
		Update("role", req.Role)

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

func generateInviteCode() string {
	return uuid.New().String()[:8]
}
