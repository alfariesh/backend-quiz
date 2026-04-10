package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/dto"
	"github.com/alfariesh/backend-quiz/internal/port"
)

var _ port.AuthServicer = (*AuthService)(nil)

type AuthService struct {
	userRepo        domain.UserRepository
	uow             domain.UnitOfWork
	jwtSecret       string
	accessDuration  time.Duration
	refreshDuration time.Duration
}

func NewAuthService(userRepo domain.UserRepository, uow domain.UnitOfWork, jwtSecret string, accessDuration, refreshDuration time.Duration) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		uow:             uow,
		jwtSecret:       jwtSecret,
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
	}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.TokenPair, *domain.User, error) {
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, domain.ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	user := &domain.User{
		Email:            req.Email,
		PasswordHash:     string(hash),
		DisplayName:      req.DisplayName,
		Timezone:         "UTC",
		DesiredRetention: 0.9,
		DailyNewLimit:    20,
		DailyReviewLimit: 200,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	tokens, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenPair, *domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	tokens, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenPair, error) {
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, domain.ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return nil, domain.ErrUnauthorized
	}

	sub, _ := claims["sub"].(string)
	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Verify user still exists
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, domain.ErrUnauthorized
	}

	return s.generateTokens(userID)
}

func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.Timezone != nil {
		user.Timezone = *req.Timezone
	}
	if req.DesiredRetention != nil {
		user.DesiredRetention = *req.DesiredRetention
	}
	if req.DailyNewLimit != nil {
		user.DailyNewLimit = *req.DailyNewLimit
	}
	if req.DailyReviewLimit != nil {
		user.DailyReviewLimit = *req.DailyReviewLimit
	}
	if req.FSRSWeights != nil {
		user.FSRSWeights = req.FSRSWeights
	}
	if req.ReminderEnabled != nil {
		user.ReminderEnabled = *req.ReminderEnabled
	}
	if req.ReminderTime != nil {
		user.ReminderTime = *req.ReminderTime
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) FindOrCreateOAuthUser(ctx context.Context, provider, providerID, email, displayName string, avatarURL *string) (*dto.TokenPair, *domain.User, error) {
	// Check if OAuth account exists
	oauthAccount, err := s.userRepo.GetOAuthAccount(ctx, provider, providerID)
	if err == nil {
		// Existing OAuth user
		user, err := s.userRepo.GetByID(ctx, oauthAccount.UserID)
		if err != nil {
			return nil, nil, err
		}
		tokens, err := s.generateTokens(user.ID)
		if err != nil {
			return nil, nil, err
		}
		return tokens, user, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, err
	}

	// Check if user with same email exists
	user, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		// Create new user + link OAuth in one transaction
		user = &domain.User{
			Email:            email,
			DisplayName:      displayName,
			Timezone:         "UTC",
			DesiredRetention: 0.9,
			DailyNewLimit:    20,
			DailyReviewLimit: 200,
		}
	} else if err != nil {
		return nil, nil, err
	}

	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if user.ID == uuid.Nil {
			if err := s.userRepo.Create(ctx, user); err != nil {
				return err
			}
		}
		return s.userRepo.CreateOAuthAccount(ctx, &domain.OAuthAccount{
			UserID:     user.ID,
			Provider:   provider,
			ProviderID: providerID,
			Email:      email,
			AvatarURL:  avatarURL,
		})
	}); err != nil {
		return nil, nil, err
	}

	tokens, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *AuthService) generateTokens(userID uuid.UUID) (*dto.TokenPair, error) {
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"sub":  userID.String(),
		"type": "access",
		"iat":  now.Unix(),
		"exp":  now.Add(s.accessDuration).Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	refreshClaims := jwt.MapClaims{
		"sub":  userID.String(),
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(s.refreshDuration).Unix(),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    now.Add(s.accessDuration).Unix(),
	}, nil
}
