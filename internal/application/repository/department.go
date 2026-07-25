package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrDepartmentNotFound = errors.New("department not found")

type departmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) interfaces.DepartmentRepository {
	return &departmentRepository{db: db}
}

func (r *departmentRepository) Create(ctx context.Context, department *types.Department) error {
	return r.db.WithContext(ctx).Create(department).Error
}

func (r *departmentRepository) GetByID(ctx context.Context, tenantID uint64, id string) (*types.Department, error) {
	var department types.Department
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&department).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDepartmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (r *departmentRepository) GetByParentAndName(ctx context.Context, tenantID uint64, parentID, name string) (*types.Department, error) {
	var department types.Department
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND parent_id = ? AND name = ?", tenantID, parentID, name).
		First(&department).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDepartmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &department, nil
}

func (r *departmentRepository) ListByTenant(ctx context.Context, tenantID uint64) ([]*types.Department, error) {
	var departments []*types.Department
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("sort_order ASC").
		Order("name ASC").
		Order("created_at ASC").
		Find(&departments).Error
	return departments, err
}

func (r *departmentRepository) Update(ctx context.Context, department *types.Department) error {
	result := r.db.WithContext(ctx).
		Model(&types.Department{}).
		Where("tenant_id = ? AND id = ?", department.TenantID, department.ID).
		Updates(map[string]interface{}{
			"parent_id":   department.ParentID,
			"name":        department.Name,
			"description": department.Description,
			"sort_order":  department.SortOrder,
			"updated_at":  department.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDepartmentNotFound
	}
	return nil
}

func (r *departmentRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&types.Department{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrDepartmentNotFound
	}
	return nil
}

func (r *departmentRepository) CountChildren(ctx context.Context, tenantID uint64, parentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&types.Department{}).
		Where("tenant_id = ? AND parent_id = ?", tenantID, parentID).
		Count(&count).Error
	return count, err
}
