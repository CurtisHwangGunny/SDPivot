package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/auth"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSDPivotLogoutHandler(t *testing.T) (*SDPivotAuthHandler, *gorm.DB, *auth.JWTManager) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.RefreshToken{}))
	manager := auth.NewJWTManager(auth.DefaultJWTConfig("focused-logout-secret"))
	return NewSDPivotAuthHandler(db, manager, nil), db, manager
}

func callSDPivotLogout(t *testing.T, handler *SDPivotAuthHandler, body []byte, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sdp/auth/logout", bytes.NewReader(body))
	if len(body) > 0 {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		c.Request.Header.Set("Authorization", "Bearer "+bearer)
	}
	handler.Logout(c)
	return w
}

func TestSDPivotLogoutAcceptsEmptyBodyAndRevokesBearerSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db, manager := newSDPivotLogoutHandler(t)
	accessToken, _, err := manager.GenerateAccessToken("user-1", types.DefaultTenantID, string(types.AccessRoleKnowledgeViewer))
	require.NoError(t, err)
	require.NoError(t, db.Create(&types.RefreshToken{
		UserID: "user-1", TokenHash: auth.HashRefreshToken("refresh-1"), Family: "family-1",
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&types.RefreshToken{
		UserID: "user-2", TokenHash: auth.HashRefreshToken("refresh-2"), Family: "family-2",
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}).Error)

	w := callSDPivotLogout(t, handler, nil, accessToken)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var userOneSessions, userTwoSessions int64
	require.NoError(t, db.Model(&types.RefreshToken{}).Where("user_id = ?", "user-1").Count(&userOneSessions).Error)
	require.NoError(t, db.Model(&types.RefreshToken{}).Where("user_id = ?", "user-2").Count(&userTwoSessions).Error)
	require.Zero(t, userOneSessions)
	require.EqualValues(t, 1, userTwoSessions)
}

func TestSDPivotLogoutRevokesRefreshTokenWithoutBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db, _ := newSDPivotLogoutHandler(t)
	require.NoError(t, db.Create(&types.RefreshToken{
		UserID: "user-1", TokenHash: auth.HashRefreshToken("refresh-1"), Family: "family-1",
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}).Error)

	w := callSDPivotLogout(t, handler, []byte(`{"refresh_token":"refresh-1"}`), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var sessions int64
	require.NoError(t, db.Model(&types.RefreshToken{}).Count(&sessions).Error)
	require.Zero(t, sessions)
}

func TestSDPivotLogoutEmptyBodyWithoutTokenIsIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _ := newSDPivotLogoutHandler(t)
	w := callSDPivotLogout(t, handler, nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
