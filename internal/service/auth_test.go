package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTokens(t *testing.T) {
	svc := &AuthService{
		jwtSecret:       "test-secret-key-123",
		accessDuration:  15 * time.Minute,
		refreshDuration: 720 * time.Hour,
	}

	userID := uuid.New()
	tokens, err := svc.generateTokens(userID)
	require.NoError(t, err)

	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Greater(t, tokens.ExpiresAt, time.Now().Unix())

	// Verify access token
	accessToken, err := jwt.Parse(tokens.AccessToken, func(t *jwt.Token) (any, error) {
		return []byte("test-secret-key-123"), nil
	})
	require.NoError(t, err)
	assert.True(t, accessToken.Valid)

	claims := accessToken.Claims.(jwt.MapClaims)
	assert.Equal(t, userID.String(), claims["sub"])
	assert.Equal(t, "access", claims["type"])

	// Verify refresh token
	refreshToken, err := jwt.Parse(tokens.RefreshToken, func(t *jwt.Token) (any, error) {
		return []byte("test-secret-key-123"), nil
	})
	require.NoError(t, err)
	assert.True(t, refreshToken.Valid)

	refreshClaims := refreshToken.Claims.(jwt.MapClaims)
	assert.Equal(t, userID.String(), refreshClaims["sub"])
	assert.Equal(t, "refresh", refreshClaims["type"])
}

func TestGenerateTokens_Expiry(t *testing.T) {
	svc := &AuthService{
		jwtSecret:       "test-secret",
		accessDuration:  15 * time.Minute,
		refreshDuration: 720 * time.Hour,
	}

	userID := uuid.New()
	tokens, err := svc.generateTokens(userID)
	require.NoError(t, err)

	// Access token should expire in ~15 minutes
	accessToken, err := jwt.Parse(tokens.AccessToken, func(t *jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)

	claims := accessToken.Claims.(jwt.MapClaims)
	exp, _ := claims.GetExpirationTime()
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), exp.Time, 5*time.Second)

	// Refresh token should expire in ~30 days
	refreshToken, err := jwt.Parse(tokens.RefreshToken, func(t *jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)

	refreshClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshExp, _ := refreshClaims.GetExpirationTime()
	assert.WithinDuration(t, time.Now().Add(720*time.Hour), refreshExp.Time, 5*time.Second)
}

func TestGenerateTokens_DifferentUsers(t *testing.T) {
	svc := &AuthService{
		jwtSecret:       "test-secret",
		accessDuration:  15 * time.Minute,
		refreshDuration: 720 * time.Hour,
	}

	user1 := uuid.New()
	user2 := uuid.New()

	tokens1, err := svc.generateTokens(user1)
	require.NoError(t, err)

	tokens2, err := svc.generateTokens(user2)
	require.NoError(t, err)

	assert.NotEqual(t, tokens1.AccessToken, tokens2.AccessToken)
	assert.NotEqual(t, tokens1.RefreshToken, tokens2.RefreshToken)
}
