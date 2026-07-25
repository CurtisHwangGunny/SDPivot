package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type DepartmentRepository interface {
	Create(ctx context.Context, department *types.Department) error
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.Department, error)
	GetByParentAndName(ctx context.Context, tenantID uint64, parentID, name string) (*types.Department, error)
	ListByTenant(ctx context.Context, tenantID uint64) ([]*types.Department, error)
	Update(ctx context.Context, department *types.Department) error
	Delete(ctx context.Context, tenantID uint64, id string) error
	CountChildren(ctx context.Context, tenantID uint64, parentID string) (int64, error)
}

type DepartmentService interface {
	Create(ctx context.Context, tenantID uint64, req *types.CreateDepartmentRequest) (*types.Department, error)
	Get(ctx context.Context, tenantID uint64, id string) (*types.Department, error)
	List(ctx context.Context, tenantID uint64) ([]*types.Department, error)
	Tree(ctx context.Context, tenantID uint64) ([]*types.DepartmentTreeNode, error)
	Update(ctx context.Context, tenantID uint64, id string, req *types.UpdateDepartmentRequest) (*types.Department, error)
	Delete(ctx context.Context, tenantID uint64, id string) error
}
