package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type lockoutUserRepo struct {
	interfaces.UserRepository
	user     *types.User
	failures int
	reset    bool
}

func (r *lockoutUserRepo) GetUserByEmail(context.Context, string) (*types.User, error) {
	return r.user, nil
}
func (r *lockoutUserRepo) RecordLoginFailure(_ context.Context, _ string, max int, duration time.Duration) (*time.Time, error) {
	r.failures++
	r.user.FailedLoginAttempts = r.failures
	if r.failures >= max {
		until := time.Now().Add(duration)
		r.user.LockedUntil = &until
		return &until, nil
	}
	return nil, nil
}
func (r *lockoutUserRepo) ResetLoginFailures(context.Context, string) error {
	r.reset = true
	return nil
}

func TestLoginLocksAccountAfterFiveFailures(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Correct1!"), bcrypt.MinCost)
	require.NoError(t, err)
	repo := &lockoutUserRepo{user: &types.User{ID: "u1", Email: "user@example.com", PasswordHash: string(hash), IsActive: true}}
	svc := &userService{userRepo: repo}
	req := &types.LoginRequest{Email: repo.user.Email, Password: "wrong"}

	for attempt := 1; attempt <= 4; attempt++ {
		response, loginErr := svc.Login(context.Background(), req)
		require.NoError(t, loginErr)
		assert.False(t, response.Success)
		assert.Equal(t, attempt, repo.failures)
	}
	response, loginErr := svc.Login(context.Background(), req)
	assert.Nil(t, response)
	assert.True(t, errors.Is(loginErr, ErrAccountLocked))
	assert.Equal(t, 5, repo.failures)

	response, loginErr = svc.Login(context.Background(), &types.LoginRequest{Email: repo.user.Email, Password: "Correct1!"})
	assert.Nil(t, response)
	assert.True(t, errors.Is(loginErr, ErrAccountLocked))
	assert.False(t, repo.reset)
}
