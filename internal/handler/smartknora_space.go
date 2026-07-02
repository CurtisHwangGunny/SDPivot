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

// SmartKnoraSpaceHandler handles knowledge space management.
type SmartKnoraSpaceHandler struct {
	db *gorm.DB
}

// NewSmartKnoraSpaceHandler creates a new space handler.
func NewSmartKnoraSpaceHandler(db *gorm.DB) *SmartKnoraSpaceHandler {
	return &SmartKnoraSpaceHandler{db: db}
}

// RegisterRoutes registers knowledge space routes.
func (h *SmartKnoraSpaceHandler) RegisterRoutes(rg *gin.RouterGroup) {
	spaces := rg.Group("/spaces")
	{
		spaces.POST("", h.CreateSpace)
		spaces.GET("", h.ListSpaces)
		spaces.GET("/:id", h.GetSpace)
		spaces.PUT("/:id", h.UpdateSpace)
		spaces.DELETE("/:id", h.DeleteSpace)
		spaces.GET("/:id/members", h.ListSpaceMembers)
		spaces.POST("/:id/members", h.AddSpaceMember)
		spaces.DELETE("/:id/members/:userId", h.RemoveSpaceMember)
		spaces.PUT("/:id/members/:userId", h.UpdateSpaceMemberRole)
	}

	categories := rg.Group("/categories")
	{
		categories.POST("", h.CreateCategory)
		categories.GET("", h.ListCategories)
		categories.DELETE("/:id", h.DeleteCategory)
	}
}

// CreateSpace creates a new knowledge space.
func (h *SmartKnoraSpaceHandler) CreateSpace(c *gin.Context) {
	var req types.CreateSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	now := time.Now()

	if req.Visibility == "" {
		req.Visibility = "team"
	}

	space := types.KnowledgeSpace{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		Icon:        req.Icon,
		CreatorID:   &userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.OrgID != "" {
		space.OrgID = &req.OrgID
	}

	if err := h.db.Create(&space).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create space"})
		return
	}

	// Add creator as owner
	member := types.SpaceMember{
		ID:        uuid.New().String(),
		SpaceID:   space.ID,
		UserID:    userID,
		Role:      "owner",
		CreatedAt: now,
	}
	h.db.Create(&member)

	c.JSON(http.StatusCreated, gin.H{"space": space})
}

// ListSpaces lists knowledge spaces the user has access to.
func (h *SmartKnoraSpaceHandler) ListSpaces(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get spaces where user is a member
	var memberSpaceIDs []string
	h.db.Model(&types.SpaceMember{}).
		Where("user_id = ?", userID).
		Pluck("space_id", &memberSpaceIDs)

	// Get all team/org visibility spaces in tenant + member-only spaces
	var spaces []types.KnowledgeSpace
	query := h.db.Where("tenant_id = ?", tenantID)
	if len(memberSpaceIDs) > 0 {
		query = query.Where("visibility IN ('team', 'org') OR id IN ?", memberSpaceIDs)
	} else {
		query = query.Where("visibility IN ('team', 'org')")
	}
	query.Order("created_at DESC").Find(&spaces)

	c.JSON(http.StatusOK, gin.H{"spaces": spaces})
}

// GetSpace gets a specific knowledge space.
func (h *SmartKnoraSpaceHandler) GetSpace(c *gin.Context) {
	spaceID := c.Param("id")

	var space types.KnowledgeSpace
	if err := h.db.Where("id = ?", spaceID).First(&space).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "space not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"space": space})
}

// UpdateSpace updates a knowledge space.
func (h *SmartKnoraSpaceHandler) UpdateSpace(c *gin.Context) {
	spaceID := c.Param("id")
	callerID := middleware.GetUserID(c)

	// Verify caller has owner/editor role on the space
	var callerMember types.SpaceMember
	if err := h.db.Where("space_id = ? AND user_id = ? AND role IN ('owner','editor')", spaceID, callerID).First(&callerMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to update this space"})
		return
	}

	var req types.UpdateSpaceRequest
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
	if req.Visibility != nil {
		updates["visibility"] = *req.Visibility
	}
	if req.Icon != nil {
		updates["icon"] = *req.Icon
	}

	h.db.Model(&types.KnowledgeSpace{}).Where("id = ?", spaceID).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "space updated"})
}

// DeleteSpace soft-deletes a knowledge space.
func (h *SmartKnoraSpaceHandler) DeleteSpace(c *gin.Context) {
	spaceID := c.Param("id")
	callerID := middleware.GetUserID(c)

	// Verify caller has owner role on the space
	var callerMember types.SpaceMember
	if err := h.db.Where("space_id = ? AND user_id = ? AND role = 'owner'", spaceID, callerID).First(&callerMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to delete this space"})
		return
	}

	h.db.Where("id = ?", spaceID).Delete(&types.KnowledgeSpace{})
	h.db.Where("space_id = ?", spaceID).Delete(&types.SpaceMember{})

	c.JSON(http.StatusOK, gin.H{"message": "space deleted"})
}

// ListSpaceMembers lists members of a knowledge space.
func (h *SmartKnoraSpaceHandler) ListSpaceMembers(c *gin.Context) {
	spaceID := c.Param("id")

	var members []types.SpaceMember
	h.db.Where("space_id = ?", spaceID).Find(&members)

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddSpaceMember adds a member to a knowledge space.
func (h *SmartKnoraSpaceHandler) AddSpaceMember(c *gin.Context) {
	spaceID := c.Param("id")

	var req types.AddSpaceMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role == "" {
		req.Role = "viewer"
	}

	member := types.SpaceMember{
		ID:        uuid.New().String(),
		SpaceID:   spaceID,
		UserID:    req.UserID,
		Role:      req.Role,
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(&member).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already a member or invalid"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"member": member})
}

// RemoveSpaceMember removes a member from a knowledge space.
func (h *SmartKnoraSpaceHandler) RemoveSpaceMember(c *gin.Context) {
	spaceID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID := middleware.GetUserID(c)

	// Verify caller has owner/editor role on the space
	var callerMember types.SpaceMember
	if err := h.db.Where("space_id = ? AND user_id = ? AND role IN ('owner','editor')", spaceID, callerID).First(&callerMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to manage space members"})
		return
	}

	h.db.Where("space_id = ? AND user_id = ?", spaceID, targetUserID).Delete(&types.SpaceMember{})
	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}

// UpdateSpaceMemberRole updates a space member's role.
func (h *SmartKnoraSpaceHandler) UpdateSpaceMemberRole(c *gin.Context) {
	spaceID := c.Param("id")
	targetUserID := c.Param("userId")
	callerID := middleware.GetUserID(c)

	// Verify caller has owner role on the space
	var callerMember types.SpaceMember
	if err := h.db.Where("space_id = ? AND user_id = ? AND role = 'owner'", spaceID, callerID).First(&callerMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to update space member roles"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=owner editor viewer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.db.Model(&types.SpaceMember{}).
		Where("space_id = ? AND user_id = ?", spaceID, targetUserID).
		Update("role", req.Role)

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// CreateCategory creates a new space category.
func (h *SmartKnoraSpaceHandler) CreateCategory(c *gin.Context) {
	var req types.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := middleware.GetTenantID(c)

	category := types.SpaceCategory{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Name:      req.Name,
		Color:     req.Color,
		CreatedAt: time.Now(),
	}
	h.db.Create(&category)

	c.JSON(http.StatusCreated, gin.H{"category": category})
}

// ListCategories lists space categories for the current tenant.
func (h *SmartKnoraSpaceHandler) ListCategories(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var categories []types.SpaceCategory
	h.db.Where("tenant_id = ?", tenantID).Order("name").Find(&categories)

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// DeleteCategory deletes a space category.
func (h *SmartKnoraSpaceHandler) DeleteCategory(c *gin.Context) {
	categoryID := c.Param("id")
	h.db.Where("id = ?", categoryID).Delete(&types.SpaceCategory{})
	c.JSON(http.StatusOK, gin.H{"message": "category deleted"})
}
