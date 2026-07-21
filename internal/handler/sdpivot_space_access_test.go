package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
)

func newSpaceAccessTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.User{}, &types.KnowledgeSpace{}, &types.SpaceMember{}))
	return db
}

func spaceTestContext(method, path, userID string, tenantID uint64, body any) (*gin.Context, *httptest.ResponseRecorder) {
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", userID)
	c.Set("tenant_id", tenantID)
	return c, w
}

func seedSpaceAccess(t *testing.T, db *gorm.DB) {
	t.Helper()
	users := []types.User{
		{ID: "owner", Username: "owner", Email: "owner@example.test", PasswordHash: "x", TenantID: 1, IsActive: true},
		{ID: "editor", Username: "editor", Email: "editor@example.test", PasswordHash: "x", TenantID: 1, IsActive: true},
		{ID: "member", Username: "member", Email: "member@example.test", PasswordHash: "x", TenantID: 1, IsActive: true},
		{ID: "viewer", Username: "viewer", Email: "viewer@example.test", PasswordHash: "x", TenantID: 1, IsActive: true},
		{ID: "other-tenant", Username: "other", Email: "other@example.test", PasswordHash: "x", TenantID: 2, IsActive: true},
	}
	require.NoError(t, db.Create(&users).Error)
	require.NoError(t, db.Create(&types.KnowledgeSpace{ID: "private", TenantID: 1, Name: "Private", Visibility: "private"}).Error)
	require.NoError(t, db.Create(&types.SpaceMember{ID: "owner-member", SpaceID: "private", UserID: "owner", Role: "owner"}).Error)
	require.NoError(t, db.Create(&types.SpaceMember{ID: "editor-member", SpaceID: "private", UserID: "editor", Role: "editor"}).Error)
	require.NoError(t, db.Create(&types.SpaceMember{ID: "viewer-member", SpaceID: "private", UserID: "viewer", Role: "viewer"}).Error)
}

func TestGetSpaceRejectsPrivateSpaceNonMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodGet, "/spaces/private", "member", 1, nil)
	c.Params = gin.Params{{Key: "id", Value: "private"}}

	h.GetSpace(c)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestAddSpaceMemberRequiresOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodPost, "/spaces/private/members", "viewer", 1, map[string]string{"user_id": "member", "role": "owner"})
	c.Params = gin.Params{{Key: "id", Value: "private"}}

	h.AddSpaceMember(c)

	require.Equal(t, http.StatusForbidden, w.Code)
	var count int64
	require.NoError(t, db.Model(&types.SpaceMember{}).Where("space_id = ? AND user_id = ?", "private", "member").Count(&count).Error)
	require.Zero(t, count)
}

func TestOwnerAddsViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodPost, "/spaces/private/members", "owner", 1, map[string]string{"user_id": "member", "role": "viewer"})
	c.Params = gin.Params{{Key: "id", Value: "private"}}

	h.AddSpaceMember(c)

	require.Equal(t, http.StatusCreated, w.Code)
	var member types.SpaceMember
	require.NoError(t, db.Where("space_id = ? AND user_id = ?", "private", "member").First(&member).Error)
	require.Equal(t, "viewer", member.Role)
}

func TestOwnerCannotAddCrossTenantUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodPost, "/spaces/private/members", "owner", 1, map[string]string{"user_id": "other-tenant", "role": "viewer"})
	c.Params = gin.Params{{Key: "id", Value: "private"}}

	h.AddSpaceMember(c)

	require.Equal(t, http.StatusForbidden, w.Code)
	var count int64
	require.NoError(t, db.Model(&types.SpaceMember{}).Where("space_id = ? AND user_id = ?", "private", "other-tenant").Count(&count).Error)
	require.Zero(t, count)
}

func TestOwnerCannotRemoveLastOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodDelete, "/spaces/private/members/owner", "owner", 1, nil)
	c.Params = gin.Params{{Key: "id", Value: "private"}, {Key: "userId", Value: "owner"}}

	h.RemoveSpaceMember(c)

	require.Equal(t, http.StatusConflict, w.Code)
	var member types.SpaceMember
	require.NoError(t, db.Where("space_id = ? AND user_id = ?", "private", "owner").First(&member).Error)
	require.Equal(t, "owner", member.Role)
}

func TestOwnerCannotDemoteLastOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodPut, "/spaces/private/members/owner", "owner", 1, map[string]string{"role": "viewer"})
	c.Params = gin.Params{{Key: "id", Value: "private"}, {Key: "userId", Value: "owner"}}

	h.UpdateSpaceMemberRole(c)

	require.Equal(t, http.StatusConflict, w.Code)
	var member types.SpaceMember
	require.NoError(t, db.Where("space_id = ? AND user_id = ?", "private", "owner").First(&member).Error)
	require.Equal(t, "owner", member.Role)
}

func TestViewerCannotUpdateSpace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodPut, "/spaces/private", "viewer", 1, map[string]string{"name": "Changed"})
	c.Params = gin.Params{{Key: "id", Value: "private"}}

	h.UpdateSpace(c)

	require.Equal(t, http.StatusForbidden, w.Code)
	var space types.KnowledgeSpace
	require.NoError(t, db.Where("id = ?", "private").First(&space).Error)
	require.Equal(t, "Private", space.Name)
}

func TestEditorAddsViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodPost, "/spaces/private/members", "editor", 1, map[string]string{"user_id": "member", "role": "viewer"})
	c.Params = gin.Params{{Key: "id", Value: "private"}}

	h.AddSpaceMember(c)

	require.Equal(t, http.StatusCreated, w.Code)
	var member types.SpaceMember
	require.NoError(t, db.Where("space_id = ? AND user_id = ?", "private", "member").First(&member).Error)
	require.Equal(t, "viewer", member.Role)
}

func TestEditorCannotAddPrivilegedMember(t *testing.T) {
	for _, role := range []string{"editor", "owner"} {
		t.Run(role, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			db := newSpaceAccessTestDB(t)
			seedSpaceAccess(t, db)
			h := NewSDPivotSpaceHandler(db)
			c, w := spaceTestContext(http.MethodPost, "/spaces/private/members", "editor", 1, map[string]string{"user_id": "member", "role": role})
			c.Params = gin.Params{{Key: "id", Value: "private"}}

			h.AddSpaceMember(c)

			require.Equal(t, http.StatusForbidden, w.Code)
			var count int64
			require.NoError(t, db.Model(&types.SpaceMember{}).Where("space_id = ? AND user_id = ?", "private", "member").Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestEditorRemovesViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newSpaceAccessTestDB(t)
	seedSpaceAccess(t, db)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodDelete, "/spaces/private/members/viewer", "editor", 1, nil)
	c.Params = gin.Params{{Key: "id", Value: "private"}, {Key: "userId", Value: "viewer"}}

	h.RemoveSpaceMember(c)

	require.Equal(t, http.StatusOK, w.Code)
	var count int64
	require.NoError(t, db.Model(&types.SpaceMember{}).Where("space_id = ? AND user_id = ?", "private", "viewer").Count(&count).Error)
	require.Zero(t, count)
}

func TestEditorCannotRemovePrivilegedMember(t *testing.T) {
	for _, target := range []string{"owner", "editor"} {
		t.Run(target, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			db := newSpaceAccessTestDB(t)
			seedSpaceAccess(t, db)
			h := NewSDPivotSpaceHandler(db)
			c, w := spaceTestContext(http.MethodDelete, "/spaces/private/members/"+target, "editor", 1, nil)
			c.Params = gin.Params{{Key: "id", Value: "private"}, {Key: "userId", Value: target}}

			h.RemoveSpaceMember(c)

			require.Equal(t, http.StatusForbidden, w.Code)
			var member types.SpaceMember
			require.NoError(t, db.Where("space_id = ? AND user_id = ?", "private", target).First(&member).Error)
			require.Contains(t, []string{"owner", "editor"}, member.Role)
		})
	}
}
