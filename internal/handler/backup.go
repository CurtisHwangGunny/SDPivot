package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type BackupHandler struct {
	service interfaces.BackupService
}

func NewBackupHandler(service interfaces.BackupService) *BackupHandler {
	return &BackupHandler{service: service}
}

func (h *BackupHandler) Create(c *gin.Context) {
	actorID, _ := types.UserIDFromContext(c.Request.Context())
	record, err := h.service.Create(c.Request.Context(), actorID, types.BackupTriggerManual)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, record)
}

func (h *BackupHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	records, total, err := h.service.List(c.Request.Context(), limit, offset)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": records, "total": total})
}

func (h *BackupHandler) Get(c *gin.Context) {
	record, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *BackupHandler) Download(c *gin.Context) {
	file, record, err := h.service.Open(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	defer file.Close()
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="`+record.FileName+`"`)
	c.Header("Cache-Control", "no-store")
	c.DataFromReader(http.StatusOK, record.SizeBytes, "application/octet-stream", file, nil)
}

func (h *BackupHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BackupHandler) Restore(c *gin.Context) {
	var req struct {
		Confirmation string `json:"confirmation" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := c.Param("id")
	if req.Confirmation != "RESTORE "+id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "confirmation must equal RESTORE " + id})
		return
	}
	ctx, cancel := contextWithRestoreTimeout(c.Request.Context())
	defer cancel()
	if err := h.service.Restore(ctx, id); err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "restored", "backup_id": id})
}

func (h *BackupHandler) GetSchedule(c *gin.Context) {
	config, err := h.service.GetSchedule(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *BackupHandler) UpdateSchedule(c *gin.Context) {
	var config types.BackupScheduleConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateSchedule(c.Request.Context(), &config); err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *BackupHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
	case errors.Is(err, service.ErrBackupBusy):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case strings.Contains(err.Error(), "required"), strings.Contains(err.Error(), "invalid"), strings.Contains(err.Error(), "not ready"), strings.Contains(err.Error(), "checksum"):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backup operation failed"})
	}
}

func contextWithRestoreTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 2*time.Hour)
}
