package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type backupContextKey struct{}

type stubBackupService struct {
	interfaces.BackupService
	create         func(context.Context, string, string) (*types.BackupRecord, error)
	list           func(context.Context, int, int) ([]*types.BackupRecord, int64, error)
	get            func(context.Context, string) (*types.BackupRecord, error)
	open           func(context.Context, string) (io.ReadCloser, *types.BackupRecord, error)
	delete         func(context.Context, string) error
	restore        func(context.Context, string) error
	getSchedule    func(context.Context) (*types.BackupScheduleConfig, error)
	updateSchedule func(context.Context, *types.BackupScheduleConfig) error
}

func (s *stubBackupService) Create(ctx context.Context, actorID, trigger string) (*types.BackupRecord, error) {
	return s.create(ctx, actorID, trigger)
}

func (s *stubBackupService) List(ctx context.Context, limit, offset int) ([]*types.BackupRecord, int64, error) {
	return s.list(ctx, limit, offset)
}

func (s *stubBackupService) Get(ctx context.Context, id string) (*types.BackupRecord, error) {
	return s.get(ctx, id)
}

func (s *stubBackupService) Open(ctx context.Context, id string) (io.ReadCloser, *types.BackupRecord, error) {
	return s.open(ctx, id)
}

func (s *stubBackupService) Delete(ctx context.Context, id string) error {
	return s.delete(ctx, id)
}

func (s *stubBackupService) Restore(ctx context.Context, id string) error {
	return s.restore(ctx, id)
}

func (s *stubBackupService) GetSchedule(ctx context.Context) (*types.BackupScheduleConfig, error) {
	return s.getSchedule(ctx)
}

func (s *stubBackupService) UpdateSchedule(ctx context.Context, config *types.BackupScheduleConfig) error {
	return s.updateSchedule(ctx, config)
}

type trackedReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackedReadCloser) Close() error {
	r.closed = true
	return nil
}

func newBackupTestRouter(svc interfaces.BackupService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewBackupHandler(svc)
	r := gin.New()
	r.POST("/backups", h.Create)
	r.GET("/backups", h.List)
	r.GET("/backups/schedule", h.GetSchedule)
	r.PUT("/backups/schedule", h.UpdateSchedule)
	r.GET("/backups/:id", h.Get)
	r.GET("/backups/:id/download", h.Download)
	r.DELETE("/backups/:id", h.Delete)
	r.POST("/backups/:id/restore", h.Restore)
	return r
}

func performBackupRequest(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody io.Reader
	if body != "" {
		requestBody = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, requestBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	ctx := context.WithValue(req.Context(), backupContextKey{}, "request-value")
	ctx = context.WithValue(ctx, types.UserIDContextKey, "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req.WithContext(ctx))
	return w
}

func requireBackupRequestContext(t *testing.T, ctx context.Context) {
	t.Helper()
	if got := ctx.Value(backupContextKey{}); got != "request-value" {
		t.Fatalf("request context was not propagated: got %v", got)
	}
}

func TestBackupHandlerCreate(t *testing.T) {
	svc := &stubBackupService{
		create: func(ctx context.Context, actorID, trigger string) (*types.BackupRecord, error) {
			requireBackupRequestContext(t, ctx)
			if actorID != "user-1" || trigger != types.BackupTriggerManual {
				t.Fatalf("unexpected create arguments: actor=%q trigger=%q", actorID, trigger)
			}
			return &types.BackupRecord{ID: "backup-1", Status: types.BackupStatusPending}, nil
		},
	}

	w := performBackupRequest(t, newBackupTestRouter(svc), http.MethodPost, "/backups", "")
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	var record types.BackupRecord
	if err := json.Unmarshal(w.Body.Bytes(), &record); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if record.ID != "backup-1" || record.Status != types.BackupStatusPending {
		t.Fatalf("unexpected response: %+v", record)
	}
}

func TestBackupHandlerList(t *testing.T) {
	svc := &stubBackupService{
		list: func(ctx context.Context, limit, offset int) ([]*types.BackupRecord, int64, error) {
			requireBackupRequestContext(t, ctx)
			if limit != 7 || offset != 14 {
				t.Fatalf("unexpected pagination: limit=%d offset=%d", limit, offset)
			}
			return []*types.BackupRecord{{ID: "backup-1"}, {ID: "backup-2"}}, 23, nil
		},
	}

	w := performBackupRequest(t, newBackupTestRouter(svc), http.MethodGet, "/backups?limit=7&offset=14", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Items []types.BackupRecord `json:"items"`
		Total int64                `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Total != 23 || len(response.Items) != 2 || response.Items[1].ID != "backup-2" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestBackupHandlerGet(t *testing.T) {
	svc := &stubBackupService{
		get: func(ctx context.Context, id string) (*types.BackupRecord, error) {
			requireBackupRequestContext(t, ctx)
			if id != "backup-7" {
				t.Fatalf("unexpected backup ID: %q", id)
			}
			return &types.BackupRecord{ID: id, Status: types.BackupStatusSucceeded}, nil
		},
	}

	w := performBackupRequest(t, newBackupTestRouter(svc), http.MethodGet, "/backups/backup-7", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"id":"backup-7"`) {
		t.Fatalf("unexpected response: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestBackupHandlerDownload(t *testing.T) {
	file := &trackedReadCloser{Reader: strings.NewReader("backup-data")}
	svc := &stubBackupService{
		open: func(ctx context.Context, id string) (io.ReadCloser, *types.BackupRecord, error) {
			requireBackupRequestContext(t, ctx)
			if id != "backup-8" {
				t.Fatalf("unexpected backup ID: %q", id)
			}
			return file, &types.BackupRecord{FileName: "weknora.dump", SizeBytes: 11}, nil
		},
	}

	w := performBackupRequest(t, newBackupTestRouter(svc), http.MethodGet, "/backups/backup-8/download", "")
	if w.Code != http.StatusOK || w.Body.String() != "backup-data" {
		t.Fatalf("unexpected download: status=%d body=%q", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="weknora.dump"` {
		t.Fatalf("unexpected content disposition: %q", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("unexpected cache control: %q", got)
	}
	if !file.closed {
		t.Fatal("download file was not closed")
	}
}

func TestBackupHandlerDelete(t *testing.T) {
	svc := &stubBackupService{
		delete: func(ctx context.Context, id string) error {
			requireBackupRequestContext(t, ctx)
			if id != "backup-9" {
				t.Fatalf("unexpected backup ID: %q", id)
			}
			return nil
		},
	}

	w := performBackupRequest(t, newBackupTestRouter(svc), http.MethodDelete, "/backups/backup-9", "")
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("unexpected response: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestBackupHandlerRestoreRequiresConfirmationAndPropagatesRequest(t *testing.T) {
	called := false
	svc := &stubBackupService{
		restore: func(ctx context.Context, id string) error {
			called = true
			requireBackupRequestContext(t, ctx)
			if id != "backup-10" {
				t.Fatalf("unexpected backup ID: %q", id)
			}
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) < time.Hour || time.Until(deadline) > 2*time.Hour {
				t.Fatalf("unexpected restore deadline: %v, present=%v", deadline, ok)
			}
			return nil
		},
	}
	r := newBackupTestRouter(svc)

	w := performBackupRequest(t, r, http.MethodPost, "/backups/backup-10/restore", `{"confirmation":"RESTORE backup-11"}`)
	if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"confirmation must equal RESTORE backup-10"}` {
		t.Fatalf("unexpected confirmation response: status=%d body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatal("restore service called for an incorrect confirmation")
	}

	w = performBackupRequest(t, r, http.MethodPost, "/backups/backup-10/restore", `{"confirmation":" RESTORE backup-10 "}`)
	if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"confirmation must equal RESTORE backup-10"}` {
		t.Fatalf("confirmation must match exactly: status=%d body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatal("restore service called for a confirmation with surrounding whitespace")
	}

	w = performBackupRequest(t, r, http.MethodPost, "/backups/backup-10/restore", `{"confirmation":"RESTORE backup-10"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !called || w.Body.String() != `{"backup_id":"backup-10","status":"restored"}` {
		t.Fatalf("unexpected restore response: called=%v body=%s", called, w.Body.String())
	}
}

func TestBackupHandlerRestoreRejectsInvalidJSON(t *testing.T) {
	svc := &stubBackupService{
		restore: func(context.Context, string) error {
			t.Fatal("restore service called for invalid JSON")
			return nil
		},
	}

	w := performBackupRequest(t, newBackupTestRouter(svc), http.MethodPost, "/backups/backup-1/restore", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestBackupHandlerSchedule(t *testing.T) {
	config := &types.BackupScheduleConfig{Enabled: true, Cron: "0 3 * * *", RetentionDays: 30}
	svc := &stubBackupService{
		getSchedule: func(ctx context.Context) (*types.BackupScheduleConfig, error) {
			requireBackupRequestContext(t, ctx)
			return config, nil
		},
		updateSchedule: func(ctx context.Context, got *types.BackupScheduleConfig) error {
			requireBackupRequestContext(t, ctx)
			if *got != (types.BackupScheduleConfig{Enabled: false, Cron: "0 1 * * 0", RetentionDays: 14}) {
				t.Fatalf("unexpected schedule update: %+v", got)
			}
			return nil
		},
	}
	r := newBackupTestRouter(svc)

	w := performBackupRequest(t, r, http.MethodGet, "/backups/schedule", "")
	if w.Code != http.StatusOK || w.Body.String() != `{"enabled":true,"cron":"0 3 * * *","retention_days":30}` {
		t.Fatalf("unexpected get schedule response: status=%d body=%s", w.Code, w.Body.String())
	}

	w = performBackupRequest(t, r, http.MethodPut, "/backups/schedule", `{"enabled":false,"cron":"0 1 * * 0","retention_days":14}`)
	if w.Code != http.StatusOK || w.Body.String() != `{"enabled":false,"cron":"0 1 * * 0","retention_days":14}` {
		t.Fatalf("unexpected update schedule response: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestBackupHandlerUpdateScheduleRejectsInvalidJSON(t *testing.T) {
	svc := &stubBackupService{
		updateSchedule: func(context.Context, *types.BackupScheduleConfig) error {
			t.Fatal("update service called for invalid JSON")
			return nil
		},
	}

	w := performBackupRequest(t, newBackupTestRouter(svc), http.MethodPut, "/backups/schedule", `{"enabled":`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestBackupHandlerErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		message    string
	}{
		{name: "not found", err: gorm.ErrRecordNotFound, statusCode: http.StatusNotFound, message: "backup not found"},
		{name: "busy", err: service.ErrBackupBusy, statusCode: http.StatusConflict, message: service.ErrBackupBusy.Error()},
		{name: "required", err: errors.New("backup file is required"), statusCode: http.StatusBadRequest, message: "backup file is required"},
		{name: "invalid", err: errors.New("invalid backup format"), statusCode: http.StatusBadRequest, message: "invalid backup format"},
		{name: "not ready", err: errors.New("backup not ready"), statusCode: http.StatusBadRequest, message: "backup not ready"},
		{name: "checksum", err: errors.New("checksum mismatch"), statusCode: http.StatusBadRequest, message: "checksum mismatch"},
		{name: "internal", err: errors.New("storage unavailable"), statusCode: http.StatusInternalServerError, message: "backup operation failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &stubBackupService{
				get: func(context.Context, string) (*types.BackupRecord, error) {
					return nil, tt.err
				},
			}
			w := performBackupRequest(t, newBackupTestRouter(svc), http.MethodGet, "/backups/backup-error", "")
			if w.Code != tt.statusCode {
				t.Fatalf("expected %d, got %d body=%s", tt.statusCode, w.Code, w.Body.String())
			}
			var response struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Error != tt.message {
				t.Fatalf("expected error %q, got %q", tt.message, response.Error)
			}
		})
	}
}
