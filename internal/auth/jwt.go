package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTConfig holds configuration for the JWT token manager.
type JWTConfig struct {
	SecretKey     string
	AccessExpiry  time.Duration // Default: 15 minutes
	RefreshExpiry time.Duration // Default: 7 days
	Issuer        string        // Default: "sdpivot-auth"
	Audience      string        // Default: "sdpivot-api"
}

// DefaultJWTConfig returns the default JWT configuration.
func DefaultJWTConfig(secretKey string) JWTConfig {
	return JWTConfig{
		SecretKey:     secretKey,
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
		Issuer:        "sdpivot-auth",
		Audience:      "sdpivot-api",
	}
}

// AccessClaims represents the JWT claims for access tokens.
type AccessClaims struct {
	jwt.RegisteredClaims
	UserID       string  `json:"sub"`
	TenantID     uint64  `json:"tenant_id"`
	Role         string  `json:"role"`
	DepartmentID *string `json:"department_id,omitempty"`
	Scope        string  `json:"scope"` // "access"
	JTI          string  `json:"jti"`
}

// JWTManager handles JWT token generation and validation.
type JWTManager struct {
	config JWTConfig
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(config JWTConfig) *JWTManager {
	return &JWTManager{config: config}
}

// GenerateAccessToken creates a signed JWT access token.
// Returns the token string and the JTI (token ID) for revocation.
func (m *JWTManager) GenerateAccessToken(userID string, tenantID uint64, role string, departmentID *string) (tokenString string, jti string, err error) {
	jti = "at_" + uuid.New().String()
	now := time.Now()

	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.AccessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
			Audience:  jwt.ClaimStrings{m.config.Audience},
			ID:        jti,
		},
		UserID:       userID,
		TenantID:     tenantID,
		Role:         role,
		DepartmentID: departmentID,
		Scope:        "access",
		JTI:          jti,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString([]byte(m.config.SecretKey))
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}
	return tokenString, jti, nil
}

// ValidateAccessToken parses and validates a JWT access token.
// Returns the claims if valid, or an error if the token is invalid/expired.
func (m *JWTManager) ValidateAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	}, jwt.WithIssuer(m.config.Issuer), jwt.WithAudience(m.config.Audience))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	if claims.Scope != "access" {
		return nil, fmt.Errorf("invalid token scope")
	}

	return claims, nil
}

// GenerateRefreshToken creates an opaque refresh token.
// Returns the raw token (to send to client) and its SHA-256 hash (to store).
func (m *JWTManager) GenerateRefreshToken() (rawToken string, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate random bytes: %w", err)
	}
	rawToken = hex.EncodeToString(b)

	hash := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(hash[:])

	return rawToken, tokenHash, nil
}

// HashRefreshToken computes the SHA-256 hash of a raw refresh token.
func HashRefreshToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

// AccessExpirySeconds returns the access token expiry in seconds.
func (m *JWTManager) AccessExpirySeconds() int64 {
	return int64(m.config.AccessExpiry.Seconds())
}

// RefreshExpiry returns the refresh token expiry duration.
func (m *JWTManager) RefreshExpiry() time.Duration {
	return m.config.RefreshExpiry
}
