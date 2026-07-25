package types

import "time"

const DefaultTagDimensionID = "00000000-0000-0000-0000-000000000001"

// TagDimension defines one of the seven platform-wide document classification axes.
type TagDimension struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Code        string    `json:"code" gorm:"type:varchar(64);not null;uniqueIndex"`
	Name        string    `json:"name" gorm:"type:varchar(128);not null"`
	Description string    `json:"description" gorm:"type:text;not null;default:''"`
	SortOrder   int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TagDimension) TableName() string { return "tag_dimensions" }

// TagDictionary is a platform-managed tag value belonging to one dimension.
type TagDictionary struct {
	ID          string        `json:"id" gorm:"type:varchar(36);primaryKey"`
	DimensionID string        `json:"dimension_id" gorm:"type:varchar(36);not null;index:idx_tag_dictionary_dimension_sort,priority:1"`
	Dimension   *TagDimension `json:"dimension,omitempty" gorm:"foreignKey:DimensionID"`
	Name        string        `json:"name" gorm:"type:varchar(128);not null;index:idx_tag_dictionary_dimension_sort,priority:3"`
	Color       string        `json:"color" gorm:"type:varchar(32);not null;default:''"`
	SortOrder   int           `json:"sort_order" gorm:"not null;default:0;index:idx_tag_dictionary_dimension_sort,priority:2"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (TagDictionary) TableName() string { return "tag_dictionary" }

// DocumentTag assigns one dictionary tag to a tenant document and records its dimension.
type DocumentTag struct {
	TenantID    uint64    `json:"tenant_id" gorm:"primaryKey;not null"`
	DocumentID  string    `json:"document_id" gorm:"type:varchar(36);primaryKey;not null"`
	TagID       string    `json:"tag_id" gorm:"type:varchar(36);primaryKey;not null"`
	DimensionID string    `json:"dimension_id" gorm:"type:varchar(36);not null"`
	CreatedAt   time.Time `json:"created_at"`
}

func (DocumentTag) TableName() string { return "document_tags" }
