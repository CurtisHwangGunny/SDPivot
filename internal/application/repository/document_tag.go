package repository

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type documentTagRepository struct {
	db *gorm.DB
}

func NewDocumentTagRepository(db *gorm.DB) interfaces.DocumentTagRepository {
	return &documentTagRepository{db: db}
}

func (r *documentTagRepository) ListClassificationDictionary(
	ctx context.Context,
) ([]*types.TagDimension, []*types.TagDictionary, error) {
	var dimensions []*types.TagDimension
	if err := r.db.WithContext(ctx).
		Order("sort_order ASC, code ASC").
		Find(&dimensions).Error; err != nil {
		return nil, nil, err
	}

	var tags []*types.TagDictionary
	if err := r.db.WithContext(ctx).
		Order("dimension_id ASC, sort_order ASC, name ASC").
		Find(&tags).Error; err != nil {
		return nil, nil, err
	}
	return dimensions, tags, nil
}

func (r *documentTagRepository) ReplaceDocumentTags(
	ctx context.Context,
	tenantID uint64,
	documentID string,
	tags []*types.DocumentTag,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&types.Knowledge{}).
			Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", documentID, tenantID).
			Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("document %s does not belong to tenant %d", documentID, tenantID)
		}

		if err := tx.Where("tenant_id = ? AND document_id = ?", tenantID, documentID).
			Delete(&types.DocumentTag{}).Error; err != nil {
			return err
		}
		if len(tags) == 0 {
			return nil
		}
		return tx.Create(&tags).Error
	})
}
