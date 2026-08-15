package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const backupScheduleKey = "backup_schedule_config"

var ErrBackupBusy = errors.New("a backup or restore operation is already running")

type backupService struct {
	db      *gorm.DB
	driver  string
	dir     string
	cron    *cron.Cron
	cronID  cron.EntryID
	mu      sync.Mutex
	started bool
	stopped bool
}

type backupSystemConfigRow struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	Key         string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Value       string `gorm:"type:text;not null;default:''"`
	Description string `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (backupSystemConfigRow) TableName() string { return "system_configs" }

func NewBackupService(db *gorm.DB) interfaces.BackupService {
	driver := strings.TrimSpace(os.Getenv("DB_DRIVER"))
	if driver == "" && db != nil && db.Dialector != nil {
		driver = db.Dialector.Name()
	}
	dir := strings.TrimSpace(os.Getenv("WEKNORA_BACKUP_DIR"))
	if dir == "" {
		if driver == "sqlite" {
			dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
			if dbPath == "" {
				dbPath = "./data/weknora.db"
			}
			dir = filepath.Join(filepath.Dir(dbPath), "backups")
		} else {
			dir = "./data/backups"
		}
	}
	return &backupService{
		db: db, driver: driver, dir: dir,
		cron: cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(cron.DefaultLogger))),
	}
}

func (s *backupService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	config, err := s.getSchedule(ctx)
	if err != nil {
		return err
	}
	if err := s.applyScheduleLocked(config); err != nil {
		return err
	}
	s.cron.Start()
	s.started = true
	logger.Infof(ctx, "[backup] service started: driver=%s dir=%s schedule_enabled=%v", s.driver, s.dir, config.Enabled)
	return nil
}

func (s *backupService) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()
	<-s.cron.Stop().Done()
}

func (s *backupService) Create(ctx context.Context, actorID, trigger string) (*types.BackupRecord, error) {
	if trigger != types.BackupTriggerScheduled {
		trigger = types.BackupTriggerManual
	}
	if !s.mu.TryLock() {
		return nil, ErrBackupBusy
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	ext := ".dump"
	format := "postgres-custom"
	if s.driver == "sqlite" {
		ext = ".db"
		format = "sqlite"
	}
	name := "weknora-" + now.Format("20060102-150405") + "-" + id[:8] + ext
	record := &types.BackupRecord{
		ID: id, Status: types.BackupStatusPending, Trigger: trigger,
		DatabaseDriver: s.driver, FileName: name, StoragePath: filepath.Join(s.dir, name),
		Format: format, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		s.mu.Unlock()
		return nil, err
	}
	go s.runDump(record.ID)
	return record, nil
}

func (s *backupService) runDump(id string) {
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), backupTimeout())
	defer cancel()
	record, err := s.Get(ctx, id)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(record).Updates(map[string]any{
		"status": types.BackupStatusRunning, "started_at": now, "updated_at": now,
	}).Error
	partial := record.StoragePath + ".partial"
	_ = os.Remove(partial)
	err = s.dump(ctx, partial)
	if err == nil {
		err = os.Rename(partial, record.StoragePath)
	}
	if err == nil {
		var size int64
		var checksum string
		size, checksum, err = backupFileInfo(record.StoragePath)
		if err == nil {
			completed := time.Now().UTC()
			err = s.db.WithContext(ctx).Model(record).Updates(map[string]any{
				"status": types.BackupStatusSucceeded, "size_bytes": size,
				"checksum_sha256": checksum, "completed_at": completed,
				"error_message": "", "updated_at": completed,
			}).Error
		}
	}
	if err != nil {
		_ = os.Remove(partial)
		completed := time.Now().UTC()
		message := sanitizeBackupError(err)
		_ = s.db.WithContext(context.Background()).Model(record).Updates(map[string]any{
			"status": types.BackupStatusFailed, "error_message": message,
			"completed_at": completed, "updated_at": completed,
		}).Error
		logger.Errorf(context.Background(), "[backup] dump %s failed: %s", id, message)
		return
	}
	logger.Infof(context.Background(), "[backup] dump %s completed", id)
	s.purgeExpired(context.Background())
}

func (s *backupService) dump(ctx context.Context, destination string) error {
	switch s.driver {
	case "postgres":
		args := []string{
			"--format=custom", "--no-owner", "--no-privileges",
			"--exclude-table=backup_records", "--file=" + destination,
			"--host=" + backupEnv("DB_HOST", "SDP_DB_HOST"), "--port=" + backupEnv("DB_PORT", "SDP_DB_PORT"),
			"--username=" + backupEnv("DB_USER", "SDP_DB_USER"), backupEnv("DB_NAME", "SDP_DB_NAME"),
		}
		cmd := exec.CommandContext(ctx, "pg_dump", args...)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+backupEnv("DB_PASSWORD", "SDP_DB_PASSWORD"))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("pg_dump: %s: %w", strings.TrimSpace(string(output)), err)
		}
		return nil
	case "sqlite":
		quoted := strings.ReplaceAll(destination, "'", "''")
		return s.db.WithContext(ctx).Exec("VACUUM INTO '" + quoted + "'").Error
	default:
		return fmt.Errorf("unsupported database driver %q", s.driver)
	}
}

func (s *backupService) List(ctx context.Context, limit, offset int) ([]*types.BackupRecord, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	if err := s.db.WithContext(ctx).Model(&types.BackupRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []*types.BackupRecord
	err := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

func (s *backupService) Get(ctx context.Context, id string) (*types.BackupRecord, error) {
	var record types.BackupRecord
	if err := s.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *backupService) Open(ctx context.Context, id string) (io.ReadCloser, *types.BackupRecord, error) {
	record, err := s.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if record.Status != types.BackupStatusSucceeded {
		return nil, nil, errors.New("backup is not ready")
	}
	file, err := os.Open(record.StoragePath)
	return file, record, err
}

func (s *backupService) Delete(ctx context.Context, id string) error {
	if !s.mu.TryLock() {
		return ErrBackupBusy
	}
	defer s.mu.Unlock()
	record, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if record.StoragePath != "" {
		if err := os.Remove(record.StoragePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return s.db.WithContext(ctx).Delete(&types.BackupRecord{}, "id = ?", id).Error
}

func (s *backupService) Restore(ctx context.Context, id string) error {
	if !s.mu.TryLock() {
		return ErrBackupBusy
	}
	defer s.mu.Unlock()
	record, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if record.Status != types.BackupStatusSucceeded {
		return errors.New("backup is not ready")
	}
	_, checksum, err := backupFileInfo(record.StoragePath)
	if err != nil {
		return err
	}
	if checksum != record.ChecksumSHA256 {
		return errors.New("backup checksum mismatch")
	}
	switch s.driver {
	case "postgres":
		args := []string{
			"--clean", "--if-exists", "--no-owner", "--no-privileges", "--exit-on-error",
			"--host=" + backupEnv("DB_HOST", "SDP_DB_HOST"), "--port=" + backupEnv("DB_PORT", "SDP_DB_PORT"),
			"--username=" + backupEnv("DB_USER", "SDP_DB_USER"), "--dbname=" + backupEnv("DB_NAME", "SDP_DB_NAME"), record.StoragePath,
		}
		cmd := exec.CommandContext(ctx, "pg_restore", args...)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+backupEnv("DB_PASSWORD", "SDP_DB_PASSWORD"))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("pg_restore: %s: %w", strings.TrimSpace(string(output)), err)
		}
		return nil
	case "sqlite":
		return s.restoreSQLite(ctx, record.StoragePath)
	default:
		return fmt.Errorf("unsupported database driver %q", s.driver)
	}
}

func (s *backupService) restoreSQLite(ctx context.Context, source string) error {
	targetDB, err := s.db.DB()
	if err != nil {
		return err
	}
	sourceDB, err := sql.Open("sqlite3", "file:"+source+"?mode=ro")
	if err != nil {
		return err
	}
	defer sourceDB.Close()
	sourceConn, err := sourceDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer sourceConn.Close()
	targetConn, err := targetDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer targetConn.Close()
	return targetConn.Raw(func(target any) error {
		targetSQLite, ok := target.(*sqlite3.SQLiteConn)
		if !ok {
			return errors.New("unexpected SQLite target connection")
		}
		return sourceConn.Raw(func(sourceRaw any) error {
			sourceSQLite, ok := sourceRaw.(*sqlite3.SQLiteConn)
			if !ok {
				return errors.New("unexpected SQLite source connection")
			}
			backup, err := targetSQLite.Backup("main", sourceSQLite, "main")
			if err != nil {
				return err
			}
			defer backup.Finish()
			for {
				done, err := backup.Step(256)
				if err != nil {
					return err
				}
				if done {
					return nil
				}
			}
		})
	})
}

func (s *backupService) GetSchedule(ctx context.Context) (*types.BackupScheduleConfig, error) {
	return s.getSchedule(ctx)
}

func (s *backupService) getSchedule(ctx context.Context) (*types.BackupScheduleConfig, error) {
	config := &types.BackupScheduleConfig{Cron: "0 0 2 * * *", RetentionDays: 30}
	var row backupSystemConfigRow
	err := s.db.WithContext(ctx).Where("key = ?", backupScheduleKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return config, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(row.Value), config); err != nil {
		return nil, fmt.Errorf("decode backup schedule: %w", err)
	}
	return config, nil
}

func (s *backupService) UpdateSchedule(ctx context.Context, config *types.BackupScheduleConfig) error {
	if config == nil {
		return errors.New("schedule config is required")
	}
	config.Cron = strings.TrimSpace(config.Cron)
	if config.Cron == "" {
		return errors.New("cron is required")
	}
	if config.RetentionDays < 0 || config.RetentionDays > 3650 {
		return errors.New("retention_days must be between 0 and 3650")
	}
	if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(config.Cron); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := backupSystemConfigRow{Key: backupScheduleKey, Value: string(encoded), Description: "Database backup schedule", CreatedAt: now, UpdatedAt: now}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{"value": row.Value, "description": row.Description, "updated_at": now}),
	}).Create(&row).Error; err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyScheduleLocked(config)
}

func (s *backupService) applyScheduleLocked(config *types.BackupScheduleConfig) error {
	if s.cronID != 0 {
		s.cron.Remove(s.cronID)
		s.cronID = 0
	}
	if !config.Enabled {
		return nil
	}
	id, err := s.cron.AddFunc(config.Cron, func() {
		if _, err := s.Create(context.Background(), "", types.BackupTriggerScheduled); err != nil && !errors.Is(err, ErrBackupBusy) {
			logger.Errorf(context.Background(), "[backup] scheduled backup failed to start: %v", err)
		}
	})
	if err != nil {
		return err
	}
	s.cronID = id
	return nil
}

func (s *backupService) purgeExpired(ctx context.Context) {
	config, err := s.getSchedule(ctx)
	if err != nil || config.RetentionDays <= 0 {
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -config.RetentionDays)
	var records []*types.BackupRecord
	if err := s.db.WithContext(ctx).Where("created_at < ?", cutoff).Find(&records).Error; err != nil {
		return
	}
	for _, record := range records {
		_ = os.Remove(record.StoragePath)
		_ = s.db.WithContext(ctx).Delete(record).Error
	}
}

func backupTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("WEKNORA_BACKUP_TIMEOUT")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return 2 * time.Hour
}

func backupFileInfo(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return 0, "", err
	}
	return size, hex.EncodeToString(hash.Sum(nil)), nil
}

func sanitizeBackupError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	for _, secret := range []string{backupEnv("DB_PASSWORD", "SDP_DB_PASSWORD"), backupEnv("DB_USER", "SDP_DB_USER"), backupEnv("DB_HOST", "SDP_DB_HOST")} {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[redacted]")
		}
	}
	const max = 1000
	if len(message) > max {
		message = message[:max]
	}
	return message
}

func backupEnv(primary, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(primary)); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(fallback))
}
