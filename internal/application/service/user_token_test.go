package service

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

type tokenTestUserRepo struct {
	interfaces.UserRepository
	user *types.User
}

func (r *tokenTestUserRepo) GetUserByID(context.Context, string) (*types.User, error) {
	return r.user, nil
}

type tokenTestRepo struct {
	interfaces.AuthTokenRepository
	record *types.AuthToken
}

func (r *tokenTestRepo) GetTokenByValue(_ context.Context, token string) (*types.AuthToken, error) {
	if r.record == nil || r.record.Token != token {
		return nil, assert.AnError
	}
	return r.record, nil
}

func TestValidateTokenRejectsInvalidExpiredAndRefreshTokens(t *testing.T) {
	user := &types.User{ID: "u1", IsActive: true}
	repo := &tokenTestRepo{}
	svc := &userService{userRepo: &tokenTestUserRepo{user: user}, tokenRepo: repo}

	_, _, err := svc.ValidateToken(context.Background(), "not-a-token")
	assert.Error(t, err)

	for name, tc := range map[string]struct {
		tokenType string
		expiry    time.Time
	}{
		"expired": {tokenType: "access", expiry: time.Now().Add(-time.Minute)},
		"refresh": {tokenType: "refresh", expiry: time.Now().Add(time.Minute)},
	} {
		t.Run(name, func(t *testing.T) {
			token, signErr := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"user_id": user.ID,
				"type":    tc.tokenType,
				"exp":     tc.expiry.Unix(),
			}).SignedString([]byte(getJwtSecret()))
			assert.NoError(t, signErr)
			repo.record = &types.AuthToken{Token: token, UserID: user.ID}
			_, _, validateErr := svc.ValidateToken(context.Background(), token)
			assert.Error(t, validateErr)
		})
	}
}
