package handler

import (
	"errors"
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
	tenantDB := middleware.TenantDB(c, h.db)
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

	if err := tenantDB.Create(&space).Error; err != nil {
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
	if err := tenantDB.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create space owner"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"space": space})
}

// ListSpaces lists knowledge spaces the user has access to.
func (h *SmartKnoraSpaceHandler) ListSpaces(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var spaces []types.KnowledgeSpace
	if err := tenantDB.Where("id IN (?)", visibleSpaceIDsQuery(tenantDB, tenantID, userID)).
		Order("created_at DESC").Find(&spaces).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list spaces"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"spaces": spaces})
}

// GetSpace gets a specific knowledge space.
func (h *SmartKnoraSpaceHandler) GetSpace(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	spaceID := c.Param("id")

	space, ok := authorizeSpace(c, tenantDB, spaceID, spaceAccessView)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"space": space})
}

// UpdateSpace updates a knowledge space.
func (h *SmartKnoraSpaceHandler) UpdateSpace(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	spaceID := c.Param("id")
	if _, ok := authorizeSpace(c, tenantDB, spaceID, spaceAccessEdit); !ok {
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

	if err := tenantDB.Model(&types.KnowledgeSpace{}).Where("id = ? AND tenant_id = ?", spaceID, middleware.GetTenantID(c)).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update space"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "space updated"})
}

// DeleteSpace soft-deletes a knowledge space.
func (h *SmartKnoraSpaceHandler) DeleteSpace(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	spaceID := c.Param("id")
	if _, ok := authorizeSpace(c, tenantDB, spaceID, spaceAccessOwner); !ok {
		return
	}

	if err := tenantDB.Where("id = ? AND tenant_id = ?", spaceID, middleware.GetTenantID(c)).Delete(&types.KnowledgeSpace{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete space"})
		return
	}
	if err := tenantDB.Where("space_id = ?", spaceID).Delete(&types.SpaceMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete space members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "space deleted"})
}

// ListSpaceMembers lists members of a knowledge space.
func (h *SmartKnoraSpaceHandler) ListSpaceMembers(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	spaceID := c.Param("id")

	if _, ok := authorizeSpace(c, tenantDB, spaceID, spaceAccessView); !ok {
		return
	}

	var members []types.SpaceMember
	if err := tenantDB.Where("space_id = ?", spaceID).Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list space members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddSpaceMember adds a member to a knowledge space.
func (h *SmartKnoraSpaceHandler) AddSpaceMember(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	spaceID := c.Param("id")

	if _, ok := authorizeSpace(c, tenantDB, spaceID, spaceAccessEdit); !ok {
		return
	}

	var callerMember types.SpaceMember
	if err := tenantDB.Where("space_id = ? AND user_id = ?", spaceID, middleware.GetUserID(c)).First(&callerMember).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load caller space membership"})
		return
	}

	var req types.AddSpaceMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role == "" {
		req.Role = "viewer"
	}
	if req.Role != "owner" && req.Role != "editor" && req.Role != "viewer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid space member role"})
		return
	}
	if callerMember.Role == "editor" && req.Role != "viewer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "editors may only add viewers"})
		return
	}

	var targetUser types.User
	if err := tenantDB.Where("id = ? AND tenant_id = ? AND is_active = ?", req.UserID, middleware.GetTenantID(c), true).First(&targetUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "target user is not an active tenant member"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate target user"})
		}
		return
	}

	var existing int64
	if err := tenantDB.Model(&types.SpaceMember{}).Where("space_id = ? AND user_id = ?", spaceID, req.UserID).Count(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check space membership"})
		return
	}
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "already a member"})
		return
	}

	member := types.SpaceMember{
		ID:        uuid.New().String(),
		SpaceID:   spaceID,
		UserID:    req.UserID,
		Role:      req.Role,
		CreatedAt: time.Now(),
	}

	if err := tenantDB.Create(&member).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already a member or invalid"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"member": member})
}

// RemoveSpaceMember removes a member from a knowledge space.
func (h *SmartKnoraSpaceHandler) RemoveSpaceMember(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	spaceID := c.Param("id")
	targetUserID := c.Param("userId")
	if _, ok := authorizeSpace(c, tenantDB, spaceID, spaceAccessEdit); !ok {
		return
	}

	var callerMember types.SpaceMember
	if err := tenantDB.Where("space_id = ? AND user_id = ?", spaceID, middleware.GetUserID(c)).First(&callerMember).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load caller space membership"})
		return
	}

	var targetMember types.SpaceMember
	if err := tenantDB.Where("space_id = ? AND user_id = ?", spaceID, targetUserID).First(&targetMember).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "space member not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load space member"})
		}
		return
	}
	if callerMember.Role == "editor" && targetMember.Role != "viewer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "editors may only remove viewers"})
		return
	}
	if targetMember.Role == "owner" {
		var ownerCount int64
		if err := tenantDB.Model(&types.SpaceMember{}).Where("space_id = ? AND role = ?", spaceID, "owner").Count(&ownerCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate space owners"})
			return
		}
		if ownerCount <= 1 {
			c.JSON(http.StatusConflict, gin.H{"error": "cannot remove the last space owner"})
			return
		}
	}
	if err := tenantDB.Delete(&targetMember).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove space member"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}

// UpdateSpaceMemberRole updates a space member's role.
func (h *SmartKnoraSpaceHandler) UpdateSpaceMemberRole(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	spaceID := c.Param("id")
	targetUserID := c.Param("userId")
	if _, ok := authorizeSpace(c, tenantDB, spaceID, spaceAccessOwner); !ok {
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=owner editor viewer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var targetMember types.SpaceMember
	if err := tenantDB.Where("space_id = ? AND user_id = ?", spaceID, targetUserID).First(&targetMember).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "space member not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load space member"})
		}
		return
	}
	if targetMember.Role == "owner" && req.Role != "owner" {
		var ownerCount int64
		if err := tenantDB.Model(&types.SpaceMember{}).Where("space_id = ? AND role = ?", spaceID, "owner").Count(&ownerCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate space owners"})
			return
		}
		if ownerCount <= 1 {
			c.JSON(http.StatusConflict, gin.H{"error": "cannot demote the last space owner"})
			return
		}
	}
	if err := tenantDB.Model(&targetMember).Update("role", req.Role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update space member role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// CreateCategory creates a new space category.
func (h *SmartKnoraSpaceHandler) CreateCategory(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
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
	tenantDB.Create(&category)

	c.JSON(http.StatusCreated, gin.H{"category": category})
}

// ListCategories lists space categories for the current tenant.
func (h *SmartKnoraSpaceHandler) ListCategories(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)

	var categories []types.SpaceCategory
	tenantDB.Where("tenant_id = ?", tenantID).Order("name").Find(&categories)

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// DeleteCategory deletes a space category.
func (h *SmartKnoraSpaceHandler) DeleteCategory(c *gin.Context) {
	tenantDB := middleware.TenantDB(c, h.db)
	categoryID := c.Param("id")
	tenantDB.Where("id = ? AND tenant_id = ?", categoryID, middleware.GetTenantID(c)).Delete(&types.SpaceCategory{})
	c.JSON(http.StatusOK, gin.H{"message": "category deleted"})
}
