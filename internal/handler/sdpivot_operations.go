package handler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/middleware"
	appRuntime "github.com/Tencent/WeKnora/internal/runtime"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const sdpivotVersionFallback = "2.0.0"

type SDPivotOperationsHandler struct {
	db            *gorm.DB
	redis         *redis.Client
	backupService interfaces.BackupService
	opMode        bool
	startedAt     time.Time
}

func NewSDPivotOperationsHandler(db *gorm.DB, redisClient *redis.Client, backupService interfaces.BackupService, opMode bool) *SDPivotOperationsHandler {
	return &SDPivotOperationsHandler{db: db, redis: redisClient, backupService: backupService, opMode: opMode, startedAt: time.Now()}
}

func (h *SDPivotOperationsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group(
		"/admin",
		middleware.RequirePermission(middleware.PermissionDepartmentManage),
		middleware.RequireSuperAdmin(),
	)
	admin.GET("/backup", h.ListBackups)
	admin.POST("/backup", h.CreateBackup)
	admin.GET("/backup/config", h.GetBackupConfig)
	admin.PUT("/backup/config", h.UpdateBackupConfig)
	admin.GET("/update/check", h.CheckUpdate)
	admin.GET("/update/logs", h.ListUpdateLogs)
	admin.GET("/update/version", h.GetVersion)
	admin.GET("/logs", h.ListLogs)
	admin.GET("/logs/levels", h.GetLogPolicy)
	admin.GET("/logs/cleaning-policy", h.GetLogPolicy)
	admin.GET("/monitor", h.GetMonitor)
}

func (h *SDPivotOperationsHandler) ListBackups(c *gin.Context) {
	page, pageSize := parsePagination(c)
	if h.backupService == nil {
		c.JSON(http.StatusOK, gin.H{"records": []any{}, "total": 0, "page": page, "page_size": pageSize, "config": backupConfigResponse(nil), "notice": "backup service is unavailable"})
		return
	}
	records, total, err := h.backupService.List(c.Request.Context(), pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}
	config, configErr := h.backupService.GetSchedule(c.Request.Context())
	response := gin.H{"records": formatBackupRecords(records), "total": total, "page": page, "page_size": pageSize, "config": backupConfigResponse(config)}
	if configErr != nil {
		response["config_notice"] = "backup schedule configuration is unavailable"
	}
	c.JSON(http.StatusOK, response)
}

func (h *SDPivotOperationsHandler) CreateBackup(c *gin.Context) {
	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backup service is unavailable"})
		return
	}
	record, err := h.backupService.Create(c.Request.Context(), middleware.GetUserID(c), types.BackupTriggerManual)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrBackupBusy) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, formatBackupRecord(record))
}

func (h *SDPivotOperationsHandler) GetBackupConfig(c *gin.Context) {
	if h.backupService == nil {
		c.JSON(http.StatusOK, backupConfigResponse(nil))
		return
	}
	config, err := h.backupService.GetSchedule(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load backup configuration"})
		return
	}
	c.JSON(http.StatusOK, backupConfigResponse(config))
}

func (h *SDPivotOperationsHandler) UpdateBackupConfig(c *gin.Context) {
	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backup service is unavailable"})
		return
	}
	var config types.BackupScheduleConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.backupService.UpdateSchedule(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, backupConfigResponse(&config))
}

func formatBackupRecords(records []*types.BackupRecord) []gin.H {
	items := make([]gin.H, 0, len(records))
	for _, record := range records {
		items = append(items, formatBackupRecord(record))
	}
	return items
}

func formatBackupRecord(record *types.BackupRecord) gin.H {
	status := record.Status
	if status == types.BackupStatusSucceeded {
		status = "success"
	} else if status == types.BackupStatusPending {
		status = types.BackupStatusRunning
	}
	remark := record.ErrorMessage
	return gin.H{
		"id": record.ID, "time": record.CreatedAt, "type": record.Trigger,
		"size": record.SizeBytes, "size_bytes": record.SizeBytes, "status": status,
		"path": record.FileName, "remark": remark, "error": record.ErrorMessage,
	}
}

func backupConfigResponse(config *types.BackupScheduleConfig) gin.H {
	if config == nil {
		config = &types.BackupScheduleConfig{Cron: "0 0 2 * * *", RetentionDays: 30}
	}
	dir := strings.TrimSpace(os.Getenv("WEKNORA_BACKUP_DIR"))
	if dir == "" {
		dir = "./data/backups"
	}
	return gin.H{
		"enabled": config.Enabled, "cron": config.Cron, "interval": config.Cron,
		"retention_days": config.RetentionDays, "retention_count": 0, "backup_directory": dir,
	}
}

func (h *SDPivotOperationsHandler) CheckUpdate(c *gin.Context) {
	current := currentSDPivotVersion()
	c.JSON(http.StatusOK, gin.H{
		"current_version": current, "latest_version": current, "has_update": false,
		"release_notes": "", "source": "no update provider configured",
	})
}

func (h *SDPivotOperationsHandler) ListUpdateLogs(c *gin.Context) {
	page, pageSize := parsePagination(c)
	if h.db == nil {
		c.JSON(http.StatusOK, gin.H{"logs": []any{}, "total": 0, "page": page, "page_size": pageSize, "notice": "update log storage is unavailable"})
		return
	}
	db := middleware.TenantDB(c, h.db)
	query := db.Model(&types.SystemUpdateLog{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count update logs"})
		return
	}
	var logs []types.SystemUpdateLog
	if err := query.Order("created_at DESC, id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list update logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total, "page": page, "page_size": pageSize})
}

func (h *SDPivotOperationsHandler) GetVersion(c *gin.Context) {
	version := currentSDPivotVersion()
	c.JSON(http.StatusOK, gin.H{
		"version": version, "backend_version": version,
		"frontend_version": envOrDefault("SDP_FRONTEND_VERSION", version),
		"op_mode":          h.opMode, "edition": Edition, "commit_id": CommitID,
		"build_time": BuildTime, "go_version": runtime.Version(),
	})
}

func currentSDPivotVersion() string {
	if value := strings.TrimSpace(os.Getenv("SDP_VERSION")); value != "" {
		return value
	}
	if Version != "" && Version != "unknown" {
		return Version
	}
	return sdpivotVersionFallback
}

type operationLogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Module  string `json:"module"`
	Message string `json:"message"`
	Source  string `json:"source"`
}

func (h *SDPivotOperationsHandler) ListLogs(c *gin.Context) {
	page, pageSize := parsePagination(c)
	path := configuredLogPath()
	if path == "" {
		c.JSON(http.StatusOK, gin.H{"logs": []operationLogEntry{}, "total": 0, "page": page, "page_size": pageSize, "notice": "no safe file log source is configured; set LOG_PATH to enable log viewing"})
		return
	}
	lines, err := readRecentLogLines(path, 10000)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"logs": []operationLogEntry{}, "total": 0, "page": page, "page_size": pageSize, "source": path, "notice": "configured log source cannot be read"})
		return
	}
	level := strings.ToUpper(strings.TrimSpace(c.Query("level")))
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))
	logs := make([]operationLogEntry, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- {
		entry := parseOperationLogLine(lines[i], path)
		if level != "" && entry.Level != level {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(entry.Message+" "+entry.Module), search) {
			continue
		}
		logs = append(logs, entry)
	}
	total := len(logs)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs[start:end], "total": total, "page": page, "page_size": pageSize, "source": path})
}

func (h *SDPivotOperationsHandler) GetLogPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"levels":          []string{"INFO", "WARN", "ERROR"},
		"cleaning_policy": gin.H{"enabled": true, "retention_days": 28, "max_size_mb": 50, "max_backups": 3},
		"source":          configuredLogPath(), "managed_by": "runtime logger configuration",
	})
}

func configuredLogPath() string {
	value := strings.TrimSpace(os.Getenv("LOG_PATH"))
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}

func readRecentLogLines(path string, maxLines int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	const maxBytes int64 = 4 << 20
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	start := info.Size() - maxBytes
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lines := make([]string, 0, maxLines)
	if start > 0 && scanner.Scan() {
		// The first line may be partial after seeking into the file.
	}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			copy(lines, lines[len(lines)-maxLines:])
			lines = lines[:maxLines]
		}
	}
	return lines, scanner.Err()
}

func parseOperationLogLine(line, source string) operationLogEntry {
	entry := operationLogEntry{Level: "INFO", Module: "backend", Message: line, Source: source}
	upper := strings.ToUpper(line)
	for _, level := range []string{"ERROR", "WARN", "INFO", "DEBUG", "FATAL"} {
		if strings.Contains(upper, level) {
			entry.Level = level
			break
		}
	}
	if len(line) >= 23 {
		if _, err := time.Parse("2006-01-02 15:04:05.000", line[:23]); err == nil {
			entry.Time = line[:23]
		}
	}
	if start := strings.Index(line, "["); start >= 0 {
		if end := strings.Index(line[start+1:], "]"); end >= 0 {
			entry.Module = line[start+1 : start+1+end]
		}
	}
	return entry
}

func (h *SDPivotOperationsHandler) GetMonitor(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	health := gin.H{"backend": serviceHealth("up", 0, "API process is running")}
	health["frontend"] = serviceHealth("up", 0, "frontend is managed outside the backend process")
	health["database"] = h.databaseHealth(ctx)
	health["postgres"] = health["database"]
	health["redis"] = h.redisHealth(ctx)
	health["queue"] = health["redis"]
	health["model_service"] = h.modelServiceHealth(c, ctx)
	c.JSON(http.StatusOK, gin.H{"health": health, "resources": collectResources(h.startedAt)})
}

func serviceHealth(status string, latency int64, detail string) gin.H {
	return gin.H{"status": status, "latency_ms": latency, "detail": detail}
}

func (h *SDPivotOperationsHandler) databaseHealth(ctx context.Context) gin.H {
	if h.db == nil {
		return serviceHealth("down", 0, "database is not configured")
	}
	sqlDB, err := h.db.DB()
	if err != nil {
		return serviceHealth("down", 0, "database handle is unavailable")
	}
	started := time.Now()
	err = sqlDB.PingContext(ctx)
	latency := time.Since(started).Milliseconds()
	if err != nil {
		return serviceHealth("down", latency, "database ping failed")
	}
	return serviceHealth("up", latency, "database ping succeeded")
}

func (h *SDPivotOperationsHandler) redisHealth(ctx context.Context) gin.H {
	if h.redis == nil {
		return serviceHealth("down", 0, "redis is not configured")
	}
	started := time.Now()
	err := h.redis.Ping(ctx).Err()
	latency := time.Since(started).Milliseconds()
	if err != nil {
		return serviceHealth("down", latency, "redis ping failed")
	}
	return serviceHealth("up", latency, "redis ping succeeded")
}

func (h *SDPivotOperationsHandler) modelServiceHealth(c *gin.Context, ctx context.Context) gin.H {
	if h.db == nil {
		return serviceHealth("down", 0, "model configuration storage is unavailable")
	}
	started := time.Now()
	var count int64
	err := middleware.TenantDB(c, h.db).WithContext(ctx).Model(&types.Model{}).Where("is_active = ?", true).Count(&count).Error
	latency := time.Since(started).Milliseconds()
	if err != nil {
		return serviceHealth("down", latency, "model configuration probe failed")
	}
	return serviceHealth("up", latency, fmt.Sprintf("%d active model configurations", count))
}

func collectResources(startedAt time.Time) gin.H {
	resources := gin.H{"cpu": gin.H{"percent": 0.0, "available": false}}
	if percent, err := cpuUsagePercent(); err == nil {
		resources["cpu"] = gin.H{"percent": percent, "available": true}
	}
	if total, used, percent, err := memoryUsage(); err == nil {
		resources["memory"] = gin.H{"total": total, "used": used, "percent": percent, "available": true}
	} else {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		resources["memory"] = gin.H{"total": stats.Sys, "used": stats.Alloc, "percent": 0.0, "available": false, "detail": "host memory is unavailable; process memory returned"}
	}
	if total, used, percent, err := diskUsage("/"); err == nil {
		resources["disk"] = gin.H{"path": "/", "total": total, "used": used, "percent": percent, "available": true}
	} else {
		resources["disk"] = gin.H{"path": "/", "total": 0, "used": 0, "percent": 0.0, "available": false}
	}
	uptime := appRuntime.ServerUptime()
	if uptime <= 0 {
		uptime = time.Since(startedAt)
	}
	resources["uptime"] = int64(uptime.Seconds())
	resources["uptime_seconds"] = int64(uptime.Seconds())
	return resources
}

func cpuUsagePercent() (float64, error) {
	total1, idle1, err := readCPUStat()
	if err != nil {
		return 0, err
	}
	time.Sleep(100 * time.Millisecond)
	total2, idle2, err := readCPUStat()
	if err != nil || total2 <= total1 {
		return 0, errors.New("cpu sample unavailable")
	}
	totalDelta := total2 - total1
	idleDelta := idle2 - idle1
	return float64(totalDelta-idleDelta) * 100 / float64(totalDelta), nil
}

func readCPUStat() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	line, _, _ := strings.Cut(string(data), "\n")
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, errors.New("invalid /proc/stat")
	}
	var total uint64
	values := make([]uint64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, err
		}
		values = append(values, value)
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return total, idle, nil
}

func memoryUsage() (uint64, uint64, float64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}
	values := map[string]uint64{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr == nil {
			values[key] = value * 1024
		}
	}
	total := values["MemTotal"]
	available := values["MemAvailable"]
	if total == 0 {
		return 0, 0, 0, errors.New("MemTotal unavailable")
	}
	used := total - available
	return total, used, float64(used) * 100 / float64(total), nil
}

func diskUsage(path string) (uint64, uint64, float64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, 0, 0, err
	}
	total := stats.Blocks * uint64(stats.Bsize)
	available := stats.Bavail * uint64(stats.Bsize)
	used := total - available
	if total == 0 {
		return 0, 0, 0, errors.New("disk size unavailable")
	}
	return total, used, float64(used) * 100 / float64(total), nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
