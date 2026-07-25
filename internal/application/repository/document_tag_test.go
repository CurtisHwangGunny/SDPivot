package repository

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const documentTagTestDDL = `
CREATE TABLE IF NOT EXISTS knowledges (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    deleted_at DATETIME
);
CREATE TABLE IF NOT EXISTS tag_dimensions (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE TABLE IF NOT EXISTS tag_dictionary (
    id VARCHAR(36) PRIMARY KEY,
    dimension_id VARCHAR(36) NOT NULL,
    name VARCHAR(128) NOT NULL,
    color VARCHAR(32) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME
);
CREATE TABLE IF NOT EXISTS document_tags (
    tenant_id INTEGER NOT NULL,
    document_id VARCHAR(36) NOT NULL,
    tag_id VARCHAR(36) NOT NULL,
    dimension_id VARCHAR(36) NOT NULL,
    confidence REAL NOT NULL DEFAULT 0,
    created_at DATETIME,
    PRIMARY KEY (tenant_id, document_id, tag_id),
    UNIQUE (tenant_id, document_id, dimension_id)
);
`

func TestReplaceDocumentTags_ReplacesAssignmentsAndConfidence(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	require.NoError(t, db.Exec(documentTagTestDDL).Error)
	repo := &documentTagRepository{db: db}
	ctx := context.Background()
	documentID := uuid.NewString()
	dimensionA, dimensionB := uuid.NewString(), uuid.NewString()
	tagA, tagB, tagC := uuid.NewString(), uuid.NewString(), uuid.NewString()
	require.NoError(t, db.Exec(
		"INSERT INTO knowledges (id, tenant_id, knowledge_base_id) VALUES (?, ?, ?)",
		documentID, 7, uuid.NewString(),
	).Error)

	require.NoError(t, repo.ReplaceDocumentTags(ctx, 7, documentID, []*types.DocumentTag{
		{TenantID: 7, DocumentID: documentID, TagID: tagA, DimensionID: dimensionA, Confidence: 0.91},
		{TenantID: 7, DocumentID: documentID, TagID: tagB, DimensionID: dimensionB, Confidence: 0.72},
	}))
	require.NoError(t, repo.ReplaceDocumentTags(ctx, 7, documentID, []*types.DocumentTag{
		{TenantID: 7, DocumentID: documentID, TagID: tagC, DimensionID: dimensionA, Confidence: 0.84},
	}))

	var rows []types.DocumentTag
	require.NoError(t, db.Order("dimension_id").Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, tagC, rows[0].TagID)
	assert.InDelta(t, 0.84, rows[0].Confidence, 0.0001)
}

func TestReplaceDocumentTags_RejectsCrossTenantDocument(t *testing.T) {
	db := setupKnowledgeTestDB(t)
	require.NoError(t, db.Exec(documentTagTestDDL).Error)
	repo := &documentTagRepository{db: db}
	documentID := uuid.NewString()
	require.NoError(t, db.Exec(
		"INSERT INTO knowledges (id, tenant_id, knowledge_base_id) VALUES (?, ?, ?)",
		documentID, 1, uuid.NewString(),
	).Error)

	err := repo.ReplaceDocumentTags(context.Background(), 2, documentID, nil)
	require.Error(t, err)
}
