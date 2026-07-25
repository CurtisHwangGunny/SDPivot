package types

import (
	"time"

	"gorm.io/gorm"
)

// Department is a tenant-scoped node in the department hierarchy.
type Department struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64         `json:"tenant_id" gorm:"not null;index:idx_departments_tenant_parent,priority:1"`
	ParentID    string         `json:"parent_id" gorm:"type:varchar(36);not null;default:'';index:idx_departments_tenant_parent,priority:2"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	Description string         `json:"description" gorm:"type:text;not null;default:''"`
	SortOrder   int            `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Department) TableName() string {
	return "departments"
}

type CreateDepartmentRequest struct {
	ParentID    string `json:"parent_id" binding:"omitempty,max=36"`
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description" binding:"omitempty,max=2000"`
	SortOrder   int    `json:"sort_order"`
}

type UpdateDepartmentRequest struct {
	ParentID    *string `json:"parent_id" binding:"omitempty,max=36"`
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	SortOrder   *int    `json:"sort_order"`
}

// DepartmentTreeNode is the recursive projection returned by the tree API.
type DepartmentTreeNode struct {
	Department
	Children []*DepartmentTreeNode `json:"children"`
}
