package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAccessTokenRejectsInvalidAndExpiredTokens(t *testing.T) {
	config := DefaultJWTConfig("phase4-secret")
	manager := NewJWTManager(config)

	valid, _, err := manager.GenerateAccessToken("u1", 7, "knowledge_viewer")
	require.NoError(t, err)
	claims, err := manager.ValidateAccessToken(valid)
	require.NoError(t, err)
	assert.Equal(t, "u1", claims.UserID)

	_, err = NewJWTManager(DefaultJWTConfig("different-secret")).ValidateAccessToken(valid)
	assert.Error(t, err)

	expiredConfig := config
	expiredConfig.AccessExpiry = -time.Minute
	expired, _, err := NewJWTManager(expiredConfig).GenerateAccessToken("u1", 7, "knowledge_viewer")
	require.NoError(t, err)
	_, err = manager.ValidateAccessToken(expired)
	assert.Error(t, err)
}

func TestValidateAccessTokenRejectsWrongScopeAndAlgorithm(t *testing.T) {
	config := DefaultJWTConfig("phase4-secret")
	manager := NewJWTManager(config)
	now := time.Now()

	wrongScope := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    config.Issuer,
			Audience:  jwt.ClaimStrings{config.Audience},
		},
		UserID: "u1", Scope: "refresh",
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongScope).SignedString([]byte(config.SecretKey))
	require.NoError(t, err)
	_, err = manager.ValidateAccessToken(token)
	assert.Error(t, err)

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, wrongScope)
	unsignedString, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = manager.ValidateAccessToken(unsignedString)
	assert.Error(t, err)
}
