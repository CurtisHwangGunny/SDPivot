package types

import "time"

type TagDimension struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Code        string    `json:"code" gorm:"type:varchar(64);not null"`
	Name        string    `json:"name" gorm:"type:varchar(128);not null"`
	Description string    `json:"description" gorm:"type:text;not null;default:''"`
	SortOrder   int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TagDimension) TableName() string { return "tag_dimensions" }

type TagDictionary struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	DimensionID string    `json:"dimension_id" gorm:"type:varchar(36);not null"`
	Name        string    `json:"name" gorm:"type:varchar(128);not null"`
	Color       string    `json:"color" gorm:"type:varchar(32);not null;default:''"`
	SortOrder   int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TagDictionary) TableName() string { return "tag_dictionary" }

type DocumentTag struct {
	TenantID    uint64    `json:"tenant_id" gorm:"primaryKey;not null"`
	DocumentID  string    `json:"document_id" gorm:"type:varchar(36);primaryKey;not null"`
	TagID       string    `json:"tag_id" gorm:"type:varchar(36);primaryKey;not null"`
	DimensionID string    `json:"dimension_id" gorm:"type:varchar(36);not null"`
	Confidence  float64   `json:"confidence" gorm:"type:decimal(5,4);not null;default:0"`
	CreatedAt   time.Time `json:"created_at"`
}

func (DocumentTag) TableName() string { return "document_tags" }

type DocumentClassificationTag struct {
	TagID         string  `json:"tag_id"`
	DimensionID   string  `json:"dimension_id"`
	DimensionCode string  `json:"dimension_code"`
	DimensionName string  `json:"dimension_name"`
	Name          string  `json:"name"`
	Color         string  `json:"color"`
	Confidence    float64 `json:"confidence"`
}
