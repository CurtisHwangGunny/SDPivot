package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Document represents an imported document in a knowledge space.
type SDPivotDocument struct {
	ID              string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64         `json:"tenant_id" gorm:"not null;index"`
	SpaceID         string         `json:"space_id" gorm:"type:varchar(36);not null;index"`
	UploaderID      string         `json:"uploader_id" gorm:"type:varchar(36)"`
	Title           string         `json:"title" gorm:"type:varchar(500);not null"`
	FileName        string         `json:"file_name" gorm:"type:varchar(255)"`
	FileType        string         `json:"file_type" gorm:"type:varchar(50)"`
	FileSize        int64          `json:"file_size"`
	FilePath        string         `json:"file_path" gorm:"type:text"`
	ContentHash     string         `json:"content_hash" gorm:"type:varchar(64)"`
	ParseStatus     string         `json:"parse_status" gorm:"type:varchar(20);not null;default:pending"`
	ChunkCount      int            `json:"chunk_count" gorm:"default:0"`
	EmbeddingStatus string         `json:"embedding_status" gorm:"type:varchar(20);not null;default:pending"`
	Version         int            `json:"version" gorm:"default:1"`
	Tags            string         `json:"tags" gorm:"type:text"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (SDPivotDocument) TableName() string { return "documents" }

func (d *SDPivotDocument) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// DocumentChunk represents a chunk of a parsed document.
type SDPivotDocumentChunk struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	DocumentID  string    `json:"document_id" gorm:"type:varchar(36);not null;index"`
	TenantID    uint64    `json:"tenant_id" gorm:"not null;index"`
	ChunkIndex  int       `json:"chunk_index" gorm:"not null"`
	Content     string    `json:"content" gorm:"type:text;not null"`
	TokenCount  int       `json:"token_count"`
	EmbeddingID string    `json:"embedding_id" gorm:"type:varchar(64)"`
	Metadata    string    `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt   time.Time `json:"created_at"`
}

func (SDPivotDocumentChunk) TableName() string { return "document_chunks" }

func (c *SDPivotDocumentChunk) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// DocumentVersion represents a version of a document.
type SDPivotDocumentVersion struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	DocumentID string    `json:"document_id" gorm:"type:varchar(36);not null;index"`
	TenantID   uint64    `json:"tenant_id" gorm:"not null;index"`
	Version    int       `json:"version" gorm:"not null"`
	FilePath   string    `json:"file_path" gorm:"type:text"`
	FileSize   int64     `json:"file_size"`
	ChunkCount int       `json:"chunk_count"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by" gorm:"type:varchar(36)"`
}

func (SDPivotDocumentVersion) TableName() string { return "document_versions" }

func (v *SDPivotDocumentVersion) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	return nil
}

// ChunkStrategy represents chunking configuration.
type SDPivotChunkStrategy struct {
	ID           string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID     uint64    `json:"tenant_id" gorm:"not null;index"`
	SpaceID      string    `json:"space_id" gorm:"type:varchar(36);index"`
	Name         string    `json:"name" gorm:"type:varchar(100);not null"`
	StrategyType string    `json:"strategy_type" gorm:"type:varchar(50);not null"`
	ChunkSize    int       `json:"chunk_size" gorm:"default:512"`
	ChunkOverlap int       `json:"chunk_overlap" gorm:"default:50"`
	SplitMarkers string    `json:"split_markers" gorm:"type:text"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (SDPivotChunkStrategy) TableName() string { return "chunk_strategies" }

func (s *SDPivotChunkStrategy) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// ─── Request/Response DTOs ────────────────────────────────────

type UploadDocumentRequest struct {
	SpaceID string `form:"space_id" binding:"required"`
	Tags    string `form:"tags"`
}

type DocumentListQuery struct {
	SpaceID     string `form:"space_id"`
	ParseStatus string `form:"parse_status"`
	Search      string `form:"search"`
	Page        int    `form:"page,default=1"`
	PageSize    int    `form:"page_size,default=20"`
}

type ChunkStrategyRequest struct {
	Name         string `json:"name" binding:"required"`
	StrategyType string `json:"strategy_type" binding:"required,oneof=fixed_size semantic markdown"`
	ChunkSize    int    `json:"chunk_size"`
	ChunkOverlap int    `json:"chunk_overlap"`
	SplitMarkers string `json:"split_markers"`
	SpaceID      string `json:"space_id"`
}

type SearchRequest struct {
	Query   string `json:"query" binding:"required"`
	SpaceID string `json:"space_id"`
	TopK    int    `json:"top_k"`
}

type SDPivotSearchResult struct {
	DocumentID string  `json:"document_id"`
	ChunkID    string  `json:"chunk_id"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	Title      string  `json:"title"`
}
