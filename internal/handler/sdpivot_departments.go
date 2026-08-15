package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

// SDPivotDepartmentHandler exposes tenant-scoped department administration APIs.
type SDPivotDepartmentHandler struct {
	*DepartmentHandler
}

func NewSDPivotDepartmentHandler(db *gorm.DB) *SDPivotDepartmentHandler {
	departmentService := service.NewDepartmentService(repository.NewDepartmentRepository(db))
	return &SDPivotDepartmentHandler{DepartmentHandler: NewDepartmentHandler(departmentService)}
}

func (h *SDPivotDepartmentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	departments := rg.Group("/admin/departments", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	departments.GET("", h.Tree)
	departments.GET("/tree", h.SelectionTree)
	departments.POST("", h.Create)
	departments.PUT("/:id", h.Update)
	departments.DELETE("/:id", h.Delete)
}

type departmentSelectionNode struct {
	ID       string                     `json:"id"`
	Name     string                     `json:"name"`
	ParentID string                     `json:"parent_id"`
	Children []*departmentSelectionNode `json:"children"`
}

func (h *SDPivotDepartmentHandler) SelectionTree(c *gin.Context) {
	tree, err := h.service.Tree(c.Request.Context(), middleware.GetTenantID(c))
	if err != nil {
		h.writeError(c, err, "failed to load department tree")
		return
	}
	if types.NormalizeAccessRole(middleware.GetRole(c)) == types.AccessRoleDepartmentAdmin {
		tree = filterDepartmentTreeScope(tree, middleware.GetDepartmentID(c))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toDepartmentSelectionTree(tree)})
}

func toDepartmentSelectionTree(tree []*types.DepartmentTreeNode) []*departmentSelectionNode {
	result := make([]*departmentSelectionNode, 0, len(tree))
	for _, node := range tree {
		result = append(result, &departmentSelectionNode{
			ID: node.ID, Name: node.Name, ParentID: node.ParentID, Children: toDepartmentSelectionTree(node.Children),
		})
	}
	return result
}

func (h *SDPivotDepartmentHandler) Tree(c *gin.Context) {
	tree, err := h.service.Tree(c.Request.Context(), middleware.GetTenantID(c))
	if err != nil {
		h.writeError(c, err, "failed to load department tree")
		return
	}
	if types.NormalizeAccessRole(middleware.GetRole(c)) == types.AccessRoleDepartmentAdmin {
		tree = filterDepartmentTreeScope(tree, middleware.GetDepartmentID(c))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tree})
}

func (h *SDPivotDepartmentHandler) Create(c *gin.Context) {
	var req types.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid department data").WithDetails(err.Error()))
		return
	}
	req.ParentID = strings.TrimSpace(req.ParentID)
	if req.ParentID == "" {
		if middleware.RequireRole(c, "super_admin") {
			return
		}
	} else if !h.canManageDepartment(c, req.ParentID) {
		return
	}

	department, err := h.service.Create(c.Request.Context(), middleware.GetTenantID(c), &req)
	if err != nil {
		h.writeError(c, err, "failed to create department")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": department})
}

func (h *SDPivotDepartmentHandler) Update(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if !h.canManageDepartment(c, id) {
		return
	}

	var req types.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid department data").WithDetails(err.Error()))
		return
	}
	if req.ParentID != nil {
		parentID := strings.TrimSpace(*req.ParentID)
		req.ParentID = &parentID
		if parentID == "" {
			if middleware.RequireRole(c, "super_admin") {
				return
			}
		} else if !h.canManageDepartment(c, parentID) {
			return
		}
	}

	department, err := h.service.Update(c.Request.Context(), middleware.GetTenantID(c), id, &req)
	if err != nil {
		h.writeError(c, err, "failed to update department")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": department})
}

func (h *SDPivotDepartmentHandler) Delete(c *gin.Context) {
	if middleware.RequireRole(c, "super_admin") {
		return
	}
	if err := h.service.Delete(c.Request.Context(), middleware.GetTenantID(c), strings.TrimSpace(c.Param("id"))); err != nil {
		h.writeError(c, err, "failed to delete department")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SDPivotDepartmentHandler) canManageDepartment(c *gin.Context, id string) bool {
	if types.NormalizeAccessRole(middleware.GetRole(c)) == types.AccessRoleSuperAdmin {
		return true
	}
	departments, err := h.service.List(c.Request.Context(), middleware.GetTenantID(c))
	if err != nil {
		h.writeError(c, err, "failed to load department scope")
		return false
	}
	for _, department := range filterDepartmentScope(departments, middleware.GetDepartmentID(c)) {
		if department.ID == id {
			return true
		}
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "department is outside the permitted scope"})
	return false
}
