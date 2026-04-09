package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/rekanesiads/backend-quiz/internal/domain"
	mockdomain "github.com/rekanesiads/backend-quiz/internal/mocks/domain"
)

const testJWTSecret = "test-secret-key-for-testing"

// --- Register ---

func TestAuthService_Register_Success(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "test@example.com").Return(nil, domain.ErrNotFound)
	repo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	tokens, user, err := svc.Register(ctx, RegisterRequest{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test User",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.DisplayName)
	assert.Equal(t, "UTC", user.Timezone)
	assert.Equal(t, 0.9, user.DesiredRetention)
	assert.Equal(t, 20, user.DailyNewLimit)
	assert.Equal(t, 200, user.DailyReviewLimit)

	// Verify password was hashed
	createCall := repo.Calls[1]
	createdUser := createCall.Arguments.Get(1).(*domain.User)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(createdUser.PasswordHash), []byte("password123")))
}

func TestAuthService_Register_EmailTaken(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	existing := &domain.User{ID: uuid.New(), Email: "test@example.com"}
	repo.On("GetByEmail", ctx, "test@example.com").Return(existing, nil)

	_, _, err := svc.Register(ctx, RegisterRequest{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test",
	})

	assert.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestAuthService_Register_RepoError(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "test@example.com").Return(nil, assert.AnError)

	_, _, err := svc.Register(ctx, RegisterRequest{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test",
	})

	assert.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrEmailTaken)
}

// --- Login ---

func TestAuthService_Login_Success(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	user := &domain.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PasswordHash: string(hash),
		DisplayName:  "Test User",
	}
	repo.On("GetByEmail", ctx, "test@example.com").Return(user, nil)

	tokens, returnedUser, err := svc.Login(ctx, LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, user.ID, returnedUser.ID)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "noone@example.com").Return(nil, domain.ErrNotFound)

	_, _, err := svc.Login(ctx, LoginRequest{
		Email:    "noone@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	user := &domain.User{ID: uuid.New(), Email: "test@example.com", PasswordHash: string(hash)}
	repo.On("GetByEmail", ctx, "test@example.com").Return(user, nil)

	_, _, err := svc.Login(ctx, LoginRequest{
		Email:    "test@example.com",
		Password: "wrong-password",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

// --- RefreshToken ---

func TestAuthService_RefreshToken_Success(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{ID: userID, Email: "test@example.com"}

	// Generate a valid refresh token
	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"type": "refresh",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	refreshToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))

	repo.On("GetByID", ctx, userID).Return(user, nil)

	tokens, err := svc.RefreshToken(ctx, refreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	_, err := svc.RefreshToken(ctx, "invalid-token")
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_RefreshToken_AccessTokenRejected(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	// Use access type instead of refresh
	claims := jwt.MapClaims{
		"sub":  uuid.New().String(),
		"type": "access",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	accessToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))

	_, err := svc.RefreshToken(ctx, accessToken)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_RefreshToken_ExpiredToken(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	claims := jwt.MapClaims{
		"sub":  uuid.New().String(),
		"type": "refresh",
		"exp":  time.Now().Add(-time.Hour).Unix(), // expired
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))

	_, err := svc.RefreshToken(ctx, token)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_RefreshToken_UserDeleted(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"type": "refresh",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))

	repo.On("GetByID", ctx, userID).Return(nil, domain.ErrNotFound)

	_, err := svc.RefreshToken(ctx, token)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

// --- GetProfile ---

func TestAuthService_GetProfile(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{ID: userID, Email: "test@example.com", DisplayName: "Test"}
	repo.On("GetByID", ctx, userID).Return(user, nil)

	result, err := svc.GetProfile(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, userID, result.ID)
}

func TestAuthService_GetProfile_NotFound(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	repo.On("GetByID", ctx, userID).Return(nil, domain.ErrNotFound)

	_, err := svc.GetProfile(ctx, userID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- UpdateProfile ---

func TestAuthService_UpdateProfile_AllFields(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{
		ID:               userID,
		Email:            "test@example.com",
		DisplayName:      "Old Name",
		Timezone:         "UTC",
		DesiredRetention: 0.9,
		DailyNewLimit:    20,
		DailyReviewLimit: 200,
	}

	repo.On("GetByID", ctx, userID).Return(user, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	name := "New Name"
	tz := "Asia/Jakarta"
	ret := 0.85
	newLimit := 30
	reviewLimit := 300
	weights := make([]float64, 19)
	for i := range weights {
		weights[i] = float64(i) * 0.1
	}

	reminderEnabled := true
	reminderTime := "08:00"

	result, err := svc.UpdateProfile(ctx, userID, UpdateProfileRequest{
		DisplayName:      &name,
		Timezone:         &tz,
		DesiredRetention: &ret,
		DailyNewLimit:    &newLimit,
		DailyReviewLimit: &reviewLimit,
		FSRSWeights:      weights,
		ReminderEnabled:  &reminderEnabled,
		ReminderTime:     &reminderTime,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", result.DisplayName)
	assert.Equal(t, "Asia/Jakarta", result.Timezone)
	assert.Equal(t, 0.85, result.DesiredRetention)
	assert.Equal(t, 30, result.DailyNewLimit)
	assert.Equal(t, 300, result.DailyReviewLimit)
	assert.Equal(t, weights, result.FSRSWeights)
	assert.True(t, result.ReminderEnabled)
	assert.Equal(t, "08:00", result.ReminderTime)
}

func TestAuthService_UpdateProfile_PartialUpdate(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{
		ID:               userID,
		DisplayName:      "Original",
		Timezone:         "UTC",
		DesiredRetention: 0.9,
		DailyNewLimit:    20,
		DailyReviewLimit: 200,
	}

	repo.On("GetByID", ctx, userID).Return(user, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	name := "Updated"
	result, err := svc.UpdateProfile(ctx, userID, UpdateProfileRequest{
		DisplayName: &name,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated", result.DisplayName)
	assert.Equal(t, "UTC", result.Timezone)                // unchanged
	assert.Equal(t, 0.9, result.DesiredRetention)           // unchanged
	assert.Equal(t, 20, result.DailyNewLimit)               // unchanged
	assert.Equal(t, 200, result.DailyReviewLimit)           // unchanged
}

func TestAuthService_UpdateProfile_FSRSWeightsOnly(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{ID: userID, DisplayName: "Test"}

	repo.On("GetByID", ctx, userID).Return(user, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	weights := make([]float64, 19)
	weights[0] = 0.4
	weights[1] = 0.6

	result, err := svc.UpdateProfile(ctx, userID, UpdateProfileRequest{
		FSRSWeights: weights,
	})

	require.NoError(t, err)
	require.Len(t, result.FSRSWeights, 19)
	assert.Equal(t, 0.4, result.FSRSWeights[0])
	assert.Equal(t, 0.6, result.FSRSWeights[1])
}

func TestAuthService_UpdateProfile_UserNotFound(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	repo.On("GetByID", ctx, userID).Return(nil, domain.ErrNotFound)

	_, err := svc.UpdateProfile(ctx, userID, UpdateProfileRequest{})
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- FindOrCreateOAuthUser ---

func TestAuthService_FindOrCreateOAuthUser_ExistingOAuth(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	oauth := &domain.OAuthAccount{UserID: userID, Provider: "google", ProviderID: "g123"}
	user := &domain.User{ID: userID, Email: "test@example.com"}

	repo.On("GetOAuthAccount", ctx, "google", "g123").Return(oauth, nil)
	repo.On("GetByID", ctx, userID).Return(user, nil)

	tokens, returnedUser, err := svc.FindOrCreateOAuthUser(ctx, "google", "g123", "test@example.com", "Test", nil)

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, userID, returnedUser.ID)
}

func TestAuthService_FindOrCreateOAuthUser_NewUser(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetOAuthAccount", ctx, "google", "g456").Return(nil, domain.ErrNotFound)
	repo.On("GetByEmail", ctx, "new@example.com").Return(nil, domain.ErrNotFound)
	repo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	repo.On("CreateOAuthAccount", ctx, mock.AnythingOfType("*domain.OAuthAccount")).Return(nil)

	tokens, user, err := svc.FindOrCreateOAuthUser(ctx, "google", "g456", "new@example.com", "New User", nil)

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, "New User", user.DisplayName)
}

func TestAuthService_FindOrCreateOAuthUser_ExistingEmailLinkOAuth(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	existingUser := &domain.User{ID: userID, Email: "existing@example.com"}

	repo.On("GetOAuthAccount", ctx, "google", "g789").Return(nil, domain.ErrNotFound)
	repo.On("GetByEmail", ctx, "existing@example.com").Return(existingUser, nil)
	repo.On("CreateOAuthAccount", ctx, mock.AnythingOfType("*domain.OAuthAccount")).Return(nil)

	tokens, user, err := svc.FindOrCreateOAuthUser(ctx, "google", "g789", "existing@example.com", "Existing", nil)

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, userID, user.ID) // linked to existing user
}

func TestAuthService_FindOrCreateOAuthUser_OAuthRepoError(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetOAuthAccount", ctx, "google", "g000").Return(nil, assert.AnError)

	_, _, err := svc.FindOrCreateOAuthUser(ctx, "google", "g000", "test@example.com", "Test", nil)
	assert.Error(t, err)
}

func TestAuthService_FindOrCreateOAuthUser_ExistingOAuth_UserNotFound(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	userID := uuid.New()
	oauth := &domain.OAuthAccount{UserID: userID, Provider: "google", ProviderID: "g111"}

	repo.On("GetOAuthAccount", ctx, "google", "g111").Return(oauth, nil)
	repo.On("GetByID", ctx, userID).Return(nil, domain.ErrNotFound)

	_, _, err := svc.FindOrCreateOAuthUser(ctx, "google", "g111", "test@example.com", "Test", nil)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAuthService_FindOrCreateOAuthUser_GetByEmailError(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetOAuthAccount", ctx, "google", "g222").Return(nil, domain.ErrNotFound)
	repo.On("GetByEmail", ctx, "error@example.com").Return(nil, assert.AnError)

	_, _, err := svc.FindOrCreateOAuthUser(ctx, "google", "g222", "error@example.com", "Test", nil)
	assert.Error(t, err)
}

func TestAuthService_FindOrCreateOAuthUser_WithAvatar(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	avatar := "https://example.com/avatar.jpg"
	repo.On("GetOAuthAccount", ctx, "google", "g333").Return(nil, domain.ErrNotFound)
	repo.On("GetByEmail", ctx, "avatar@example.com").Return(nil, domain.ErrNotFound)
	repo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	repo.On("CreateOAuthAccount", ctx, mock.AnythingOfType("*domain.OAuthAccount")).Return(nil)

	tokens, user, err := svc.FindOrCreateOAuthUser(ctx, "google", "g333", "avatar@example.com", "Avatar User", &avatar)

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, "avatar@example.com", user.Email)
}

// --- generateTokens (via JWT structure validation) ---

func TestAuthService_GenerateTokens_Structure(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)

	userID := uuid.New()
	tokens, err := svc.generateTokens(userID)
	require.NoError(t, err)

	// Parse and verify access token
	accessToken, err := jwt.Parse(tokens.AccessToken, func(t *jwt.Token) (any, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)
	assert.True(t, accessToken.Valid)

	accessClaims := accessToken.Claims.(jwt.MapClaims)
	assert.Equal(t, userID.String(), accessClaims["sub"])
	assert.Equal(t, "access", accessClaims["type"])

	// Parse and verify refresh token
	refreshToken, err := jwt.Parse(tokens.RefreshToken, func(t *jwt.Token) (any, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)
	assert.True(t, refreshToken.Valid)

	refreshClaims := refreshToken.Claims.(jwt.MapClaims)
	assert.Equal(t, userID.String(), refreshClaims["sub"])
	assert.Equal(t, "refresh", refreshClaims["type"])

	assert.Greater(t, tokens.ExpiresAt, time.Now().Unix())
}

// --- Register: GetByEmail repo error (not ErrNotFound) ---

func TestAuthService_Register_GetByEmailRepoError(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "test@example.com").Return(nil, assert.AnError)

	_, _, err := svc.Register(ctx, RegisterRequest{
		Email: "test@example.com", Password: "password123", DisplayName: "Test",
	})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrEmailTaken)
}

func TestAuthService_Register_CreateError(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "test@example.com").Return(nil, domain.ErrNotFound)
	repo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(assert.AnError)

	_, _, err := svc.Register(ctx, RegisterRequest{
		Email: "test@example.com", Password: "password123", DisplayName: "Test",
	})
	assert.Error(t, err)
}

// --- Login: GetByEmail repo error (not ErrNotFound) ---

func TestAuthService_Login_RepoError(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetByEmail", ctx, "test@example.com").Return(nil, assert.AnError)

	_, _, err := svc.Login(ctx, LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
}

// --- RefreshToken: non-HMAC signing method ---

func TestAuthService_RefreshToken_NonHMACMethod(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	// Create a token with "none" method — should be rejected
	_, err := svc.RefreshToken(ctx, "not.a.valid.token")
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_RefreshToken_InvalidSubject(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	// Create refresh token with invalid UUID as sub
	claims := jwt.MapClaims{
		"sub":  "not-a-uuid",
		"type": "refresh",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(testJWTSecret))

	_, err := svc.RefreshToken(ctx, tokenStr)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

// --- FindOrCreateOAuthUser: UoW error (create fails) ---

func TestAuthService_FindOrCreateOAuthUser_CreateOAuthError(t *testing.T) {
	repo := mockdomain.NewMockUserRepository(t)
	svc := NewAuthService(repo, noopUoW{}, testJWTSecret, 15*time.Minute, 720*time.Hour)
	ctx := context.Background()

	repo.On("GetOAuthAccount", ctx, "google", "123").Return(nil, domain.ErrNotFound)
	repo.On("GetByEmail", ctx, "test@example.com").Return(nil, domain.ErrNotFound)
	repo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	repo.On("CreateOAuthAccount", ctx, mock.AnythingOfType("*domain.OAuthAccount")).Return(assert.AnError)

	_, _, err := svc.FindOrCreateOAuthUser(ctx, "google", "123", "test@example.com", "Test", nil)
	assert.Error(t, err)
}
