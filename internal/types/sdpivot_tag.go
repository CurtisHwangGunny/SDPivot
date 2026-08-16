package types

import (
	"time"

	"gorm.io/gorm"
)

type SDPivotTagDimension struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    *uint64        `json:"tenant_id,omitempty"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
	SortOrder   int            `json:"sort_order"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-"`
}

func (SDPivotTagDimension) TableName() string { return "tag_dimensions" }

type SDPivotDocumentTag struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64         `json:"tenant_id"`
	DocumentID  string         `json:"document_id"`
	DimensionID string         `json:"dimension_id"`
	TagID       string         `json:"tag_id"`
	Source      string         `json:"source"`
	Confidence  float64        `json:"confidence"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-"`
}

func (SDPivotDocumentTag) TableName() string { return "document_tags" }

type SDPivotTagAdjustmentLog struct {
	ID         string    `json:"id"`
	TenantID   uint64    `json:"tenant_id"`
	DocumentID string    `json:"document_id"`
	OldTagID   *string   `json:"old_tag_id"`
	NewTagID   *string   `json:"new_tag_id"`
	Operator   string    `json:"operator"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func (SDPivotTagAdjustmentLog) TableName() string { return "tag_adjustment_log" }

type SDPivotTagFeedback struct {
	ID          string     `json:"id"`
	TenantID    uint64     `json:"tenant_id"`
	DocumentID  string     `json:"document_id"`
	TagID       string     `json:"tag_id"`
	OriginalTag string     `json:"original_tag"`
	Feedback    string     `json:"feedback"`
	Status      string     `json:"status"`
	CreatedBy   string     `json:"created_by"`
	ReviewedBy  *string    `json:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (SDPivotTagFeedback) TableName() string { return "tag_feedback_queue" }
