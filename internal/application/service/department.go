package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

var (
	ErrDepartmentNotFound = errors.New("department not found")
	ErrDepartmentConflict = errors.New("a department with this name already exists under the selected parent")
	ErrDepartmentCycle    = errors.New("a department cannot be moved into itself or its descendants")
	ErrDepartmentNotEmpty = errors.New("department has child departments")
	ErrDepartmentName     = errors.New("department name is required")
)

type departmentService struct {
	repo interfaces.DepartmentRepository
}

func NewDepartmentService(repo interfaces.DepartmentRepository) interfaces.DepartmentService {
	return &departmentService{repo: repo}
}

func normalizeDepartmentName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrDepartmentName
	}
	return name, nil
}

func (s *departmentService) ensureUnique(ctx context.Context, tenantID uint64, parentID, name, excludeID string) error {
	existing, err := s.repo.GetByParentAndName(ctx, tenantID, parentID, name)
	if err == nil {
		if existing.ID != excludeID {
			return ErrDepartmentConflict
		}
		return nil
	}
	if errors.Is(err, repository.ErrDepartmentNotFound) {
		return nil
	}
	return err
}

func (s *departmentService) Create(ctx context.Context, tenantID uint64, req *types.CreateDepartmentRequest) (*types.Department, error) {
	name, err := normalizeDepartmentName(req.Name)
	if err != nil {
		return nil, err
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID != "" {
		if _, err := s.repo.GetByID(ctx, tenantID, parentID); err != nil {
			if errors.Is(err, repository.ErrDepartmentNotFound) {
				return nil, ErrDepartmentNotFound
			}
			return nil, err
		}
	}
	if err := s.ensureUnique(ctx, tenantID, parentID, name, ""); err != nil {
		return nil, err
	}
	now := time.Now()
	department := &types.Department{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		ParentID:    parentID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		SortOrder:   req.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, department); err != nil {
		return nil, err
	}
	return department, nil
}

func (s *departmentService) Get(ctx context.Context, tenantID uint64, id string) (*types.Department, error) {
	department, err := s.repo.GetByID(ctx, tenantID, id)
	if errors.Is(err, repository.ErrDepartmentNotFound) {
		return nil, ErrDepartmentNotFound
	}
	return department, err
}

func (s *departmentService) List(ctx context.Context, tenantID uint64) ([]*types.Department, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *departmentService) Tree(ctx context.Context, tenantID uint64) ([]*types.DepartmentTreeNode, error) {
	departments, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	nodes := make(map[string]*types.DepartmentTreeNode, len(departments))
	for _, department := range departments {
		nodes[department.ID] = &types.DepartmentTreeNode{
			Department: *department,
			Children:   make([]*types.DepartmentTreeNode, 0),
		}
	}
	roots := make([]*types.DepartmentTreeNode, 0)
	for _, department := range departments {
		node := nodes[department.ID]
		if parent, ok := nodes[department.ParentID]; department.ParentID != "" && ok {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	return roots, nil
}

func (s *departmentService) Update(ctx context.Context, tenantID uint64, id string, req *types.UpdateDepartmentRequest) (*types.Department, error) {
	department, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		department.Name, err = normalizeDepartmentName(*req.Name)
		if err != nil {
			return nil, err
		}
	}
	if req.ParentID != nil {
		parentID := strings.TrimSpace(*req.ParentID)
		if parentID == id {
			return nil, ErrDepartmentCycle
		}
		for currentID := parentID; currentID != ""; {
			parent, parentErr := s.repo.GetByID(ctx, tenantID, currentID)
			if errors.Is(parentErr, repository.ErrDepartmentNotFound) {
				return nil, ErrDepartmentNotFound
			}
			if parentErr != nil {
				return nil, parentErr
			}
			if parent.ID == id {
				return nil, ErrDepartmentCycle
			}
			currentID = parent.ParentID
		}
		department.ParentID = parentID
	}
	if req.Description != nil {
		department.Description = strings.TrimSpace(*req.Description)
	}
	if req.SortOrder != nil {
		department.SortOrder = *req.SortOrder
	}
	if err := s.ensureUnique(ctx, tenantID, department.ParentID, department.Name, department.ID); err != nil {
		return nil, err
	}
	department.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, department); err != nil {
		if errors.Is(err, repository.ErrDepartmentNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}
	return department, nil
}

func (s *departmentService) Delete(ctx context.Context, tenantID uint64, id string) error {
	if _, err := s.Get(ctx, tenantID, id); err != nil {
		return err
	}
	count, err := s.repo.CountChildren(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrDepartmentNotEmpty
	}
	if err := s.repo.Delete(ctx, tenantID, id); errors.Is(err, repository.ErrDepartmentNotFound) {
		return ErrDepartmentNotFound
	} else {
		return err
	}
}
