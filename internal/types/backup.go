package types

import "time"

const (
	BackupStatusPending   = "pending"
	BackupStatusRunning   = "running"
	BackupStatusSucceeded = "succeeded"
	BackupStatusFailed    = "failed"

	BackupTriggerManual    = "manual"
	BackupTriggerScheduled = "scheduled"
)

// BackupRecord tracks a database dump managed by the platform backup service.
// StoragePath is intentionally omitted from JSON because it is an internal
// server path; clients address artifacts only by record ID.
type BackupRecord struct {
	ID             string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	Status         string     `json:"status" gorm:"type:varchar(20);not null;index"`
	Trigger        string     `json:"trigger" gorm:"type:varchar(20);not null"`
	DatabaseDriver string     `json:"database_driver" gorm:"type:varchar(20);not null"`
	FileName       string     `json:"file_name" gorm:"type:varchar(255);not null;default:''"`
	StoragePath    string     `json:"-" gorm:"type:text;not null;default:''"`
	Format         string     `json:"format" gorm:"type:varchar(20);not null;default:''"`
	SizeBytes      int64      `json:"size_bytes" gorm:"not null;default:0"`
	ChecksumSHA256 string     `json:"checksum_sha256" gorm:"type:char(64);not null;default:''"`
	CreatedBy      string     `json:"created_by" gorm:"type:varchar(36);not null;default:''"`
	ErrorMessage   string     `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (BackupRecord) TableName() string { return "backup_records" }

type BackupScheduleConfig struct {
	Enabled       bool   `json:"enabled"`
	Cron          string `json:"cron"`
	RetentionDays int    `json:"retention_days"`
}
