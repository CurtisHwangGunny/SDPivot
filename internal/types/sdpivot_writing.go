package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WritingDraft is the persisted user-owned document produced by the writing assistant.
type WritingDraft struct {
	ID               string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID           string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	TenantID         uint64    `json:"tenant_id" gorm:"not null;index"`
	Title            string    `json:"title" gorm:"type:varchar(500)"`
	Category         string    `json:"category" gorm:"type:varchar(100)"`
	Content          string    `json:"content" gorm:"type:text"`
	SpaceID          string    `json:"space_id" gorm:"type:varchar(36)"`
	SourceType       string    `json:"source_type" gorm:"type:varchar(30);not null;default:knowledge_base"`
	WebSearchEnabled bool      `json:"web_search_enabled" gorm:"not null;default:false"`
	Status           string    `json:"status" gorm:"type:varchar(20);default:draft"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (WritingDraft) TableName() string { return "writing_drafts" }

type WriteCategoryConfig struct {
	ID               string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64    `json:"tenant_id" gorm:"not null;index"`
	Category         string    `json:"category" gorm:"type:varchar(50);not null;index"`
	DefaultSpaceID   string    `json:"default_space_id" gorm:"type:varchar(36)"`
	SourceType       string    `json:"source_type" gorm:"type:varchar(30);not null;default:knowledge_base"`
	WebSearchEnabled bool      `json:"web_search_enabled" gorm:"not null;default:false"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (WriteCategoryConfig) TableName() string { return "write_category_config" }
func (c *WriteCategoryConfig) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}

// SDPivotDocumentChunk maps the OP bootstrap document_chunks table. It is kept
// separate from the core Chunk model because the SDPivot document pipeline has
// its own documents table and space ACL semantics.
type SDPivotDocumentChunk struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	DocumentID string    `json:"document_id" gorm:"type:varchar(36);not null;index"`
	TenantID   uint64    `json:"tenant_id" gorm:"not null;index"`
	ChunkIndex int       `json:"chunk_index" gorm:"not null"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	CreatedAt  time.Time `json:"created_at"`
}

func (SDPivotDocumentChunk) TableName() string { return "document_chunks" }
func (c *SDPivotDocumentChunk) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}
