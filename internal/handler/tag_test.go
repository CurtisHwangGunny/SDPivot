package handler

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubTagService struct {
	interfaces.KnowledgeTagService
	list   func(context.Context, string, *types.Pagination, string) (*types.PageResult, error)
	create func(context.Context, string, string, string, int) (*types.KnowledgeTag, error)
	update func(context.Context, string, *string, *string, *int) (*types.KnowledgeTag, error)
	delete func(context.Context, string, bool, bool, []string) error
}

func (s *stubTagService) ListTags(
	ctx context.Context, kbID string, page *types.Pagination, keyword string,
) (*types.PageResult, error) {
	return s.list(ctx, kbID, page, keyword)
}

func (s *stubTagService) CreateTag(
	ctx context.Context, kbID, name, color string, sortOrder int,
) (*types.KnowledgeTag, error) {
	return s.create(ctx, kbID, name, color, sortOrder)
}

func (s *stubTagService) UpdateTag(
	ctx context.Context, id string, name, color *string, sortOrder *int,
) (*types.KnowledgeTag, error) {
	return s.update(ctx, id, name, color, sortOrder)
}

func (s *stubTagService) DeleteTag(
	ctx context.Context, id string, force, contentOnly bool, excludeIDs []string,
) error {
	return s.delete(ctx, id, force, contentOnly, excludeIDs)
}

type stubTagRepository struct {
	interfaces.KnowledgeTagRepository
	getBySeqID func(context.Context, uint64, int64) (*types.KnowledgeTag, error)
}

func (r *stubTagRepository) GetBySeqID(
	ctx context.Context, tenantID uint64, seqID int64,
) (*types.KnowledgeTag, error) {
	return r.getBySeqID(ctx, tenantID, seqID)
}

type stubTagChunkRepository struct {
	interfaces.ChunkRepository
	listBySeqID func(context.Context, uint64, []int64) ([]*types.Chunk, error)
}

func (r *stubTagChunkRepository) ListChunksBySeqID(
	ctx context.Context, tenantID uint64, seqIDs []int64,
) ([]*types.Chunk, error) {
	return r.listBySeqID(ctx, tenantID, seqIDs)
}

func newTagHandlerTestRouter(
	tenantID uint64,
	service interfaces.KnowledgeTagService,
	tagRepo interfaces.KnowledgeTagRepository,
	chunkRepo interfaces.ChunkRepository,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.Use(middleware.ErrorHandler())
	h := NewTagHandler(service, tagRepo, chunkRepo)
	r.GET("/knowledge-bases/:id/tags", h.ListTags)
	r.POST("/knowledge-bases/:id/tags", h.CreateTag)
	r.PUT("/knowledge-bases/:id/tags/:tag_id", h.UpdateTag)
	r.DELETE("/knowledge-bases/:id/tags/:tag_id", h.DeleteTag)
	return r
}

func performTagRequest(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

func assertTagError(t *testing.T, w *httptest.ResponseRecorder, status int, code apperrors.ErrorCode) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("expected status %d, got %d body=%s", status, w.Code, w.Body.String())
	}
	var response struct {
		Success bool `json:"success"`
		Error   struct {
			Code apperrors.ErrorCode `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Success || response.Error.Code != code {
		t.Fatalf("unexpected error response: %+v body=%s", response, w.Body.String())
	}
}

func TestTagHandlerListPassesTenantPaginationAndKeyword(t *testing.T) {
	want := &types.PageResult{Total: 1, Page: 3, PageSize: 25, Data: []*types.KnowledgeTag{{ID: "tag-1"}}}
	service := &stubTagService{
		list: func(ctx context.Context, kbID string, page *types.Pagination, keyword string) (*types.PageResult, error) {
			if tenantID, ok := types.TenantIDFromContext(ctx); !ok || tenantID != 41 {
				t.Fatalf("expected tenant 41 in service context, got %d present=%v", tenantID, ok)
			}
			if kbID != "kb-1" || page.Page != 3 || page.PageSize != 25 || keyword != "finance" {
				t.Fatalf("unexpected list arguments: kb=%q page=%+v keyword=%q", kbID, page, keyword)
			}
			return want, nil
		},
	}
	r := newTagHandlerTestRouter(41, service, nil, nil)
	w := performTagRequest(t, r, http.MethodGet, "/knowledge-bases/kb-1/tags?page=3&page_size=25&keyword=finance", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Success bool              `json:"success"`
		Data    *types.PageResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if !response.Success || response.Data == nil || response.Data.Total != want.Total {
		t.Fatalf("unexpected list response: %+v", response)
	}
}

func TestTagHandlerCreatePassesPayload(t *testing.T) {
	service := &stubTagService{
		create: func(ctx context.Context, kbID, name, color string, sortOrder int) (*types.KnowledgeTag, error) {
			if tenantID, ok := types.TenantIDFromContext(ctx); !ok || tenantID != 42 {
				t.Fatalf("expected tenant 42 in service context, got %d present=%v", tenantID, ok)
			}
			if kbID != "kb-create" || name != "Priority" || color != "#123456" || sortOrder != 7 {
				t.Fatalf("unexpected create arguments: kb=%q name=%q color=%q sort=%d", kbID, name, color, sortOrder)
			}
			return &types.KnowledgeTag{ID: "created-tag", Name: name, Color: color, SortOrder: sortOrder}, nil
		},
	}
	r := newTagHandlerTestRouter(42, service, nil, nil)
	w := performTagRequest(t, r, http.MethodPost, "/knowledge-bases/kb-create/tags", `{"name":"Priority","color":"#123456","sort_order":7}`)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"id":"created-tag"`)) {
		t.Fatalf("unexpected create response: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestTagHandlerUpdateResolvesUUIDAndPassesPointers(t *testing.T) {
	service := &stubTagService{
		update: func(_ context.Context, id string, name, color *string, sortOrder *int) (*types.KnowledgeTag, error) {
			if id != "tag-uuid" {
				t.Fatalf("expected UUID to pass through, got %q", id)
			}
			if name == nil || *name != "Renamed" || color != nil || sortOrder == nil || *sortOrder != 0 {
				t.Fatalf("unexpected update payload: name=%v color=%v sort=%v", name, color, sortOrder)
			}
			return &types.KnowledgeTag{ID: id, Name: *name, SortOrder: *sortOrder}, nil
		},
	}
	r := newTagHandlerTestRouter(43, service, nil, nil)
	w := performTagRequest(t, r, http.MethodPut, "/knowledge-bases/kb-1/tags/tag-uuid", `{"name":"Renamed","sort_order":0}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTagHandlerUpdateResolvesNumericSeqIDWithTenant(t *testing.T) {
	repo := &stubTagRepository{
		getBySeqID: func(ctx context.Context, tenantID uint64, seqID int64) (*types.KnowledgeTag, error) {
			if contextTenant, ok := types.TenantIDFromContext(ctx); !ok || contextTenant != 44 {
				t.Fatalf("expected tenant 44 in repository context, got %d present=%v", contextTenant, ok)
			}
			if tenantID != 44 || seqID != 91 {
				t.Fatalf("unexpected sequence lookup: tenant=%d seq=%d", tenantID, seqID)
			}
			return &types.KnowledgeTag{ID: "resolved-uuid", SeqID: seqID}, nil
		},
	}
	service := &stubTagService{
		update: func(_ context.Context, id string, name, color *string, sortOrder *int) (*types.KnowledgeTag, error) {
			if id != "resolved-uuid" || name != nil || color == nil || *color != "blue" || sortOrder != nil {
				t.Fatalf("unexpected resolved update arguments: id=%q name=%v color=%v sort=%v", id, name, color, sortOrder)
			}
			return &types.KnowledgeTag{ID: id, Color: *color}, nil
		},
	}
	r := newTagHandlerTestRouter(44, service, repo, nil)
	w := performTagRequest(t, r, http.MethodPut, "/knowledge-bases/kb-1/tags/91", `{"color":"blue"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTagHandlerDeletePassesFlagsAndConvertsExcludeSeqIDs(t *testing.T) {
	repo := &stubTagRepository{
		getBySeqID: func(_ context.Context, tenantID uint64, seqID int64) (*types.KnowledgeTag, error) {
			if tenantID != 45 || seqID != 17 {
				t.Fatalf("unexpected tag lookup: tenant=%d seq=%d", tenantID, seqID)
			}
			return &types.KnowledgeTag{ID: "delete-uuid"}, nil
		},
	}
	chunkRepo := &stubTagChunkRepository{
		listBySeqID: func(ctx context.Context, tenantID uint64, seqIDs []int64) ([]*types.Chunk, error) {
			if contextTenant, ok := types.TenantIDFromContext(ctx); !ok || contextTenant != 45 {
				t.Fatalf("expected tenant 45 in chunk context, got %d present=%v", contextTenant, ok)
			}
			if tenantID != 45 || !reflect.DeepEqual(seqIDs, []int64{101, 102}) {
				t.Fatalf("unexpected exclusion lookup: tenant=%d seq_ids=%v", tenantID, seqIDs)
			}
			return []*types.Chunk{{ID: "chunk-a"}, {ID: "chunk-b"}}, nil
		},
	}
	service := &stubTagService{
		delete: func(ctx context.Context, id string, force, contentOnly bool, excludeIDs []string) error {
			if tenantID, ok := types.TenantIDFromContext(ctx); !ok || tenantID != 45 {
				t.Fatalf("expected tenant 45 in service context, got %d present=%v", tenantID, ok)
			}
			if id != "delete-uuid" || !force || !contentOnly || !reflect.DeepEqual(excludeIDs, []string{"chunk-a", "chunk-b"}) {
				t.Fatalf("unexpected delete arguments: id=%q force=%v content_only=%v exclude=%v", id, force, contentOnly, excludeIDs)
			}
			return nil
		},
	}
	r := newTagHandlerTestRouter(45, service, repo, chunkRepo)
	w := performTagRequest(t, r, http.MethodDelete, "/knowledge-bases/kb-1/tags/17?force=true&content_only=true", `{"exclude_ids":[101,102]}`)
	if w.Code != http.StatusOK || w.Body.String() != `{"success":true}` {
		t.Fatalf("unexpected delete response: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestTagHandlerInvalidInputUsesProductionErrorEnvelope(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "pagination", method: http.MethodGet, path: "/knowledge-bases/kb-1/tags?page=0"},
		{name: "create body", method: http.MethodPost, path: "/knowledge-bases/kb-1/tags", body: `{"color":"red"}`},
		{name: "update body", method: http.MethodPut, path: "/knowledge-bases/kb-1/tags/tag-uuid", body: `{"sort_order":"first"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTagHandlerTestRouter(46, &stubTagService{}, nil, nil)
			w := performTagRequest(t, r, tt.method, tt.path, tt.body)
			assertTagError(t, w, http.StatusBadRequest, apperrors.ErrBadRequest)
		})
	}
}

func TestTagHandlerErrorsUseProductionErrorEnvelope(t *testing.T) {
	t.Run("numeric tag not found", func(t *testing.T) {
		repo := &stubTagRepository{
			getBySeqID: func(_ context.Context, tenantID uint64, seqID int64) (*types.KnowledgeTag, error) {
				if tenantID != 47 || seqID != 404 {
					t.Fatalf("unexpected lookup: tenant=%d seq=%d", tenantID, seqID)
				}
				return nil, stderrors.New("missing")
			},
		}
		r := newTagHandlerTestRouter(47, &stubTagService{}, repo, nil)
		w := performTagRequest(t, r, http.MethodPut, "/knowledge-bases/kb-1/tags/404", `{}`)
		assertTagError(t, w, http.StatusNotFound, apperrors.ErrNotFound)
	})

	t.Run("service error", func(t *testing.T) {
		service := &stubTagService{
			list: func(context.Context, string, *types.Pagination, string) (*types.PageResult, error) {
				return nil, apperrors.NewBadRequestError("list rejected")
			},
		}
		r := newTagHandlerTestRouter(48, service, nil, nil)
		w := performTagRequest(t, r, http.MethodGet, "/knowledge-bases/kb-1/tags", "")
		assertTagError(t, w, http.StatusBadRequest, apperrors.ErrBadRequest)
	})
}
