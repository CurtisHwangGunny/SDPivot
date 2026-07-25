package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	service interfaces.DepartmentService
}

func NewDepartmentHandler(service interfaces.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{service: service}
}

func (h *DepartmentHandler) List(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	departments, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		h.writeError(c, err, "failed to list departments")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": departments})
}

func (h *DepartmentHandler) Tree(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	tree, err := h.service.Tree(c.Request.Context(), tenantID)
	if err != nil {
		h.writeError(c, err, "failed to load department tree")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tree})
}

func (h *DepartmentHandler) Get(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	department, err := h.service.Get(c.Request.Context(), tenantID, strings.TrimSpace(c.Param("department_id")))
	if err != nil {
		h.writeError(c, err, "failed to get department")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": department})
}

func (h *DepartmentHandler) Create(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	var req types.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid department data").WithDetails(err.Error()))
		return
	}
	department, err := h.service.Create(c.Request.Context(), tenantID, &req)
	if err != nil {
		h.writeError(c, err, "failed to create department")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": department})
}

func (h *DepartmentHandler) Update(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	var req types.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid department data").WithDetails(err.Error()))
		return
	}
	department, err := h.service.Update(c.Request.Context(), tenantID, strings.TrimSpace(c.Param("department_id")), &req)
	if err != nil {
		h.writeError(c, err, "failed to update department")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": department})
}

func (h *DepartmentHandler) Delete(c *gin.Context) {
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), tenantID, strings.TrimSpace(c.Param("department_id"))); err != nil {
		h.writeError(c, err, "failed to delete department")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *DepartmentHandler) writeError(c *gin.Context, err error, fallback string) {
	logger.Errorf(c.Request.Context(), "%s: %v", fallback, err)
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound):
		c.Error(apperrors.NewNotFoundError("department not found"))
	case errors.Is(err, service.ErrDepartmentConflict):
		c.Error(apperrors.NewConflictError(err.Error()))
	case errors.Is(err, service.ErrDepartmentCycle), errors.Is(err, service.ErrDepartmentName):
		c.Error(apperrors.NewValidationError(err.Error()))
	case errors.Is(err, service.ErrDepartmentNotEmpty):
		c.Error(apperrors.NewConflictError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(fallback).WithDetails(err.Error()))
	}
}
