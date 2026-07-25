package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type apiTokenTestUserRepo struct {
	interfaces.UserRepository
	user *types.User
}

func (r *apiTokenTestUserRepo) GetUserByID(context.Context, string) (*types.User, error) {
	return r.user, nil
}

type apiTokenTestRepo struct {
	interfaces.APITokenRepository
	record  *types.APIToken
	touched bool
	revoked bool
}

func (r *apiTokenTestRepo) Create(_ context.Context, token *types.APIToken) error {
	r.record = token
	return nil
}

func (r *apiTokenTestRepo) GetByHash(_ context.Context, hash string) (*types.APIToken, error) {
	if r.record == nil || r.record.TokenHash != hash {
		return nil, assert.AnError
	}
	return r.record, nil
}

func (r *apiTokenTestRepo) Touch(context.Context, string, time.Time) error {
	r.touched = true
	return nil
}

func (r *apiTokenTestRepo) RevokeByHash(_ context.Context, hash string, at time.Time) error {
	if r.record == nil || r.record.TokenHash != hash {
		return assert.AnError
	}
	r.record.RevokedAt = &at
	r.revoked = true
	return nil
}

func TestAPITokenLifecycle(t *testing.T) {
	repo := &apiTokenTestRepo{}
	svc := &userService{
		userRepo:     &apiTokenTestUserRepo{user: &types.User{ID: "u1", TenantID: 7, IsActive: true}},
		apiTokenRepo: repo,
	}
	ctx := context.WithValue(context.Background(), types.UserContextKey, &types.User{ID: "u1", TenantID: 7})
	ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(9))

	raw, record, err := svc.CreateAPIToken(ctx, "test-cli")
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	assert.Contains(t, raw, "wkn_")
	assert.NotEqual(t, raw, record.TokenHash)
	assert.Equal(t, apiTokenHash(raw), record.TokenHash)
	assert.Equal(t, uint64(9), record.TenantID)

	user, tenantID, err := svc.ValidateAPIToken(context.Background(), raw)
	require.NoError(t, err)
	assert.Equal(t, "u1", user.ID)
	assert.Equal(t, uint64(9), tenantID)
	assert.True(t, repo.touched)

	require.NoError(t, svc.RevokeAPIToken(context.Background(), raw))
	assert.True(t, repo.revoked)
	_, _, err = svc.ValidateAPIToken(context.Background(), raw)
	require.Error(t, err)
}
