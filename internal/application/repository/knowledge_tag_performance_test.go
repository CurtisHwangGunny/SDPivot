package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const tagFilterQueryBaseline = time.Second

func setupKnowledgeTagPerformanceDB(tb testing.TB, documentCount int) (*gorm.DB, string, types.KnowledgeListFilter) {
	tb.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		tb.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		tb.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	tb.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.Exec(knowledgeTagTestDDL).Error; err != nil {
		tb.Fatal(err)
	}
	for _, statement := range []string{
		"CREATE INDEX idx_knowledges_tenant_kb ON knowledges (tenant_id, knowledge_base_id)",
		"CREATE INDEX idx_document_tags_document ON document_tags (tenant_id, document_id)",
		"CREATE INDEX idx_document_tags_tag ON document_tags (tenant_id, tag_id)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			tb.Fatal(err)
		}
	}

	const kbID = "performance-kb"
	const departmentID = "department"
	const lifecycleID = "lifecycle"
	tx := db.Begin()
	if tx.Error != nil {
		tb.Fatal(tx.Error)
	}
	for i := 0; i < documentCount; i++ {
		documentID := fmt.Sprintf("document-%06d", i)
		if err := tx.Exec(`
			INSERT INTO knowledges (id, tenant_id, knowledge_base_id, type, title, parse_status)
			VALUES (?, 1, ?, 'file', ?, 'completed')
		`, documentID, kbID, documentID).Error; err != nil {
			tb.Fatal(err)
		}
		departmentTag := "legal"
		if i%2 == 1 {
			departmentTag = "finance"
		}
		lifecycleTag := "active"
		if i%3 == 0 {
			lifecycleTag = "draft"
		}
		if err := tx.Exec(`
			INSERT INTO document_tags (tenant_id, document_id, tag_id, dimension_id, confidence)
			VALUES (1, ?, ?, ?, 0.95), (1, ?, ?, ?, 0.95)
		`, documentID, departmentTag, departmentID, documentID, lifecycleTag, lifecycleID).Error; err != nil {
			tb.Fatal(err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		tb.Fatal(err)
	}

	filter := types.KnowledgeListFilter{DimensionTagFilters: []types.DimensionTagFilter{
		{DimensionID: departmentID, TagIDs: []string{"legal", "finance"}},
		{DimensionID: lifecycleID, TagIDs: []string{"active"}},
	}}
	return db, kbID, filter
}

func queryKnowledgeTagPerformanceBaseline(
	ctx context.Context,
	db *gorm.DB,
	kbID string,
	filter types.KnowledgeListFilter,
) (int, error) {
	query := db.WithContext(ctx).Model(&types.Knowledge{}).
		Where("tenant_id = ? AND knowledge_base_id = ?", uint64(1), kbID)
	query = applyKnowledgeListFilter(query, filter)
	var ids []string
	if err := query.Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	return len(ids), nil
}

func TestKnowledgeTagFilterPerformanceBaseline(t *testing.T) {
	db, kbID, filter := setupKnowledgeTagPerformanceDB(t, 5000)
	started := time.Now()
	count, err := queryKnowledgeTagPerformanceBaseline(context.Background(), db, kbID, filter)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3333 {
		t.Fatalf("matched %d documents, want 3333", count)
	}
	if elapsed := time.Since(started); elapsed >= tagFilterQueryBaseline {
		t.Fatalf("tag filter query took %s, baseline is <%s", elapsed, tagFilterQueryBaseline)
	}
}

func BenchmarkKnowledgeTagFilterQuery(b *testing.B) {
	db, kbID, filter := setupKnowledgeTagPerformanceDB(b, 5000)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		count, err := queryKnowledgeTagPerformanceBaseline(ctx, db, kbID, filter)
		if err != nil {
			b.Fatal(err)
		}
		if count != 3333 {
			b.Fatalf("matched %d documents, want 3333", count)
		}
	}
}
