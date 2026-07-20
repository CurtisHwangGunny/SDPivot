package handler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
)

func newSmartKnoraLLMTestService(t *testing.T) (*SmartKnoraLLMService, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:smartknora-llm-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&types.Model{}); err != nil {
		t.Fatalf("migrate model: %v", err)
	}
	return &SmartKnoraLLMService{db: db}, db
}

func addSmartKnoraModel(t *testing.T, db *gorm.DB, model types.Model) {
	t.Helper()
	if model.Name == "" {
		model.Name = model.ID
	}
	if model.Type == "" {
		model.Type = types.ModelTypeKnowledgeQA
	}
	if model.Source == "" {
		model.Source = types.ModelSourceRemote
	}
	if model.Status == "" {
		model.Status = types.ModelStatusActive
	}
	if err := db.Create(&model).Error; err != nil {
		t.Fatalf("create model %s: %v", model.ID, err)
	}
}

func TestSmartKnoraFindChatModelPreference(t *testing.T) {
	svc, db := newSmartKnoraLLMTestService(t)
	addSmartKnoraModel(t, db, types.Model{ID: "builtin-default", TenantID: 99999, IsBuiltin: true, IsDefault: true})
	addSmartKnoraModel(t, db, types.Model{ID: "tenant-fallback", TenantID: 42})
	addSmartKnoraModel(t, db, types.Model{ID: "tenant-default", TenantID: 42, IsDefault: true})

	model, err := svc.findChatModel(context.Background(), 42, "")
	if err != nil {
		t.Fatalf("find default: %v", err)
	}
	if model.ID != "tenant-default" {
		t.Fatalf("expected tenant-default, got %s", model.ID)
	}
}

func TestSmartKnoraFindChatModelBuiltinFallback(t *testing.T) {
	svc, db := newSmartKnoraLLMTestService(t)
	addSmartKnoraModel(t, db, types.Model{ID: "builtin-default", TenantID: 99999, IsBuiltin: true, IsDefault: true})
	model, err := svc.findChatModel(context.Background(), 42, "")
	if err != nil {
		t.Fatalf("find builtin default: %v", err)
	}
	if model.ID != "builtin-default" {
		t.Fatalf("expected builtin-default, got %s", model.ID)
	}
}

func TestSmartKnoraFindChatModelRejectsInvisibleOrInvalidModels(t *testing.T) {
	svc, db := newSmartKnoraLLMTestService(t)
	addSmartKnoraModel(t, db, types.Model{ID: "other-tenant", TenantID: 77})
	addSmartKnoraModel(t, db, types.Model{ID: "inactive", TenantID: 42, Status: types.ModelStatus("inactive")})
	addSmartKnoraModel(t, db, types.Model{ID: "embedding", TenantID: 42, Type: types.ModelTypeEmbedding})

	for _, id := range []string{"other-tenant", "inactive", "embedding", "missing"} {
		if _, err := svc.findChatModel(context.Background(), 42, id); !errors.Is(err, ErrSmartKnoraLLMModelNotAvailable) {
			t.Fatalf("model %s: expected unavailable, got %v", id, err)
		}
	}
}

func TestSmartKnoraFindChatModelExplicitDoesNotFallback(t *testing.T) {
	svc, db := newSmartKnoraLLMTestService(t)
	addSmartKnoraModel(t, db, types.Model{ID: "tenant-default", TenantID: 42, IsDefault: true})
	if _, err := svc.findChatModel(context.Background(), 42, "missing"); !errors.Is(err, ErrSmartKnoraLLMModelNotAvailable) {
		t.Fatalf("expected explicit missing model to fail without fallback, got %v", err)
	}
}
