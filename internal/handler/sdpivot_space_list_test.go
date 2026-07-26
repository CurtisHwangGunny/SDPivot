package handler

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestListSpacesAdminGetsAllTenantSpaces(t *testing.T) {
	db := newSpaceAccessTestDB(t)
	now := time.Now()
	ownerID := "canonical-owner"
	creatorID := "legacy-creator"
	require.NoError(t, db.Create(&[]types.KnowledgeSpace{
		{ID: "tenant-private", TenantID: 1, Name: "Tenant private", Visibility: "private", OwnerID: &ownerID, CreatedAt: now},
		{ID: "legacy-private", TenantID: 1, Name: "Legacy private", Visibility: "private", CreatorID: &creatorID, CreatedAt: now.Add(-time.Minute)},
		{ID: "other-tenant", TenantID: 2, Name: "Other tenant", Visibility: "team", CreatedAt: now.Add(time.Minute)},
	}).Error)

	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodGet, "/spaces", "admin", 1, nil)
	c.Set("role", string(types.AccessRoleSuperAdmin))
	h.ListSpaces(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var response struct {
		Spaces []types.KnowledgeSpace `json:"spaces"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Len(t, response.Spaces, 2)
	require.Equal(t, "tenant-private", response.Spaces[0].ID)
	require.Equal(t, ownerID, *response.Spaces[0].OwnerID)
	require.Equal(t, "legacy-private", response.Spaces[1].ID)
	require.Equal(t, creatorID, *response.Spaces[1].OwnerID)
}

func TestListSpacesAuthorizedUserGetsVisibleAndMemberSpaces(t *testing.T) {
	db := newSpaceAccessTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&[]types.KnowledgeSpace{
		{ID: "team", TenantID: 1, Name: "Team", Visibility: "team", CreatedAt: now},
		{ID: "member-private", TenantID: 1, Name: "Member private", Visibility: "private", CreatedAt: now.Add(-time.Minute)},
		{ID: "hidden-private", TenantID: 1, Name: "Hidden private", Visibility: "private", CreatedAt: now.Add(-2 * time.Minute)},
		{ID: "other-tenant", TenantID: 2, Name: "Other tenant", Visibility: "team", CreatedAt: now.Add(time.Minute)},
	}).Error)
	require.NoError(t, db.Create(&types.SpaceMember{ID: "member-link", SpaceID: "member-private", UserID: "member", Role: "viewer"}).Error)

	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodGet, "/spaces", "member", 1, nil)
	c.Set("role", string(types.AccessRoleKnowledgeViewer))
	h.ListSpaces(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var response struct {
		Spaces []types.KnowledgeSpace `json:"spaces"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, []string{"team", "member-private"}, []string{response.Spaces[0].ID, response.Spaces[1].ID})
}

func TestListSpacesReturnsEmptyArray(t *testing.T) {
	db := newSpaceAccessTestDB(t)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodGet, "/spaces", "member", 1, nil)
	c.Set("role", string(types.AccessRoleKnowledgeViewer))
	h.ListSpaces(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"spaces":[]}`, w.Body.String())
}
