package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestCreateSpaceCreatesCanonicalSpaceAndOwner(t *testing.T) {
	db := newSpaceAccessTestDB(t)
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodPost, "/spaces", "creator", types.DefaultTenantID, map[string]string{
		"name":       "Created space",
		"visibility": "private",
	})
	c.Set("role", string(types.AccessRoleKnowledgeEditor))

	h.CreateSpace(c)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var space types.KnowledgeSpace
	require.NoError(t, db.Where("name = ?", "Created space").First(&space).Error)
	require.Equal(t, types.DefaultTenantID, space.TenantID)
	require.Equal(t, "creator", requireStringPointer(t, space.OwnerID))
	require.Equal(t, "creator", requireStringPointer(t, space.CreatorID))

	var owner types.SpaceMember
	require.NoError(t, db.Where("space_id = ? AND user_id = ?", space.ID, "creator").First(&owner).Error)
	require.Equal(t, "owner", owner.Role)

	getContext, getResponse := spaceTestContext(http.MethodGet, "/spaces/"+space.ID, "creator", types.DefaultTenantID, nil)
	getContext.Params = gin.Params{{Key: "id", Value: space.ID}}
	h.GetSpace(getContext)
	require.Equal(t, http.StatusOK, getResponse.Code, getResponse.Body.String())
}

func TestCreateSpaceRollsBackWhenOwnerCreationFails(t *testing.T) {
	db := newSpaceAccessTestDB(t)
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail_space_owner", func(tx *gorm.DB) {
		if tx.Statement.Table == "space_members" {
			tx.AddError(errors.New("forced owner creation failure"))
		}
	}))
	h := NewSDPivotSpaceHandler(db)
	c, w := spaceTestContext(http.MethodPost, "/spaces", "creator", types.DefaultTenantID, map[string]string{"name": "Rolled back space"})
	c.Set("role", string(types.AccessRoleKnowledgeEditor))

	h.CreateSpace(c)

	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	require.JSONEq(t, `{"error":"failed to create space owner"}`, w.Body.String())
	var spaceCount int64
	require.NoError(t, db.Model(&types.KnowledgeSpace{}).Where("name = ?", "Rolled back space").Count(&spaceCount).Error)
	require.Zero(t, spaceCount)
	var memberCount int64
	require.NoError(t, db.Model(&types.SpaceMember{}).Count(&memberCount).Error)
	require.Zero(t, memberCount)
}

func TestCreateSpaceAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name     string
		role     string
		tenantID uint64
	}{
		{name: "viewer cannot create", role: string(types.AccessRoleKnowledgeViewer), tenantID: types.DefaultTenantID},
		{name: "noncanonical tenant cannot create", role: string(types.AccessRoleKnowledgeEditor), tenantID: types.DefaultTenantID + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newSpaceAccessTestDB(t)
			h := NewSDPivotSpaceHandler(db)
			c, w := spaceTestContext(http.MethodPost, "/spaces", "creator", tc.tenantID, map[string]string{"name": "Forbidden space"})
			c.Set("role", tc.role)

			h.CreateSpace(c)

			require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
			var count int64
			require.NoError(t, db.Model(&types.KnowledgeSpace{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func requireStringPointer(t *testing.T, value *string) string {
	t.Helper()
	require.NotNil(t, value)
	return *value
}
