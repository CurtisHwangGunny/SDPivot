package types

import "time"

// SystemUpdateLog records an SDPivot deployment update or rollback attempt.
type SystemUpdateLog struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64    `json:"-" gorm:"not null;index"`
	FromVersion string    `json:"from_version" gorm:"type:varchar(64);not null;default:''"`
	ToVersion   string    `json:"to_version" gorm:"type:varchar(64);not null;default:''"`
	Status      string    `json:"status" gorm:"type:varchar(20);not null;default:'success';index"`
	Type        string    `json:"type" gorm:"column:update_type;type:varchar(20);not null;default:'online'"`
	Content     string    `json:"content" gorm:"type:text;not null;default:''"`
	CreatedBy   string    `json:"created_by,omitempty" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt   time.Time `json:"time" gorm:"column:created_at"`
}

func (SystemUpdateLog) TableName() string { return "system_update_log" }
