package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/dto"
	"github.com/alfariesh/backend-quiz/internal/port"
	"github.com/alfariesh/backend-quiz/pkg/mailer"
	"github.com/alfariesh/backend-quiz/pkg/password"
)

var _ port.AuthServicer = (*AuthService)(nil)

type AuthPolicy struct {
	OTPLength                 int
	OTPTTL                    time.Duration
	OTPMaxAttempts            int
	ResetTokenTTL             time.Duration
	LoginLockoutWindow        time.Duration
	LoginLockoutMaxFails      int
	RequireEmailVerified      bool
	AccountDeletionGracePeriod time.Duration
}

type AuthService struct {
	userRepo        domain.UserRepository
	evRepo          domain.EmailVerificationRepository
	prtRepo         domain.PasswordResetTokenRepository
	rtRepo          domain.RefreshTokenRepository
	laRepo          domain.LoginAttemptRepository
	auditRepo       domain.AuthAuditLogRepository
	exportRepo      domain.UserDataExportRepository
	uow             domain.UnitOfWork
	mailer          mailer.Mailer
	passwordPolicy  password.Policy
	jwtSecret       string
	accessDuration  time.Duration
	refreshDuration time.Duration
	frontendURL     string
	policy          AuthPolicy
	now             func() time.Time
}

func NewAuthService(
	userRepo domain.UserRepository,
	evRepo domain.EmailVerificationRepository,
	prtRepo domain.PasswordResetTokenRepository,
	rtRepo domain.RefreshTokenRepository,
	laRepo domain.LoginAttemptRepository,
	auditRepo domain.AuthAuditLogRepository,
	exportRepo domain.UserDataExportRepository,
	uow domain.UnitOfWork,
	mail mailer.Mailer,
	passwordPolicy password.Policy,
	jwtSecret string,
	accessDuration, refreshDuration time.Duration,
	frontendURL string,
	policy AuthPolicy,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		evRepo:          evRepo,
		prtRepo:         prtRepo,
		rtRepo:          rtRepo,
		laRepo:          laRepo,
		auditRepo:       auditRepo,
		exportRepo:      exportRepo,
		uow:             uow,
		mailer:          mail,
		passwordPolicy:  passwordPolicy,
		jwtSecret:       jwtSecret,
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
		frontendURL:     frontendURL,
		policy:          policy,
		now:             time.Now,
	}
}

func (s *AuthService) validatePassword(ctx context.Context, pwd string) error {
	if err := s.passwordPolicy.Validate(ctx, pwd); err != nil {
		switch {
		case errors.Is(err, password.ErrCompromised):
			return domain.ErrPasswordCompromised
		case errors.Is(err, password.ErrTooShort), errors.Is(err, password.ErrTooLong), errors.Is(err, password.ErrTooWeak):
			return domain.ErrPasswordTooWeak
		default:
			return err
		}
	}
	return nil
}

func (s *AuthService) emitAudit(ctx context.Context, event domain.AuthAuditEvent, userID *uuid.UUID, meta RequestMeta, metadata map[string]any) {
	if s.auditRepo == nil {
		return
	}
	if err := s.auditRepo.Create(ctx, &domain.AuthAuditLog{
		UserID:    userID,
		Event:     event,
		IPAddress: meta.IPAddress,
		UserAgent: meta.UserAgent,
		Metadata:  metadata,
	}); err != nil {
		slog.WarnContext(ctx, "failed to emit audit log",
			slog.String("event", string(event)),
			slog.String("error", err.Error()),
		)
	}
}

// ── Registration & login ──────────────────────────────────────

type RequestMeta struct {
	UserAgent string
	IPAddress string
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.TokenPair, *domain.User, error) {
	return s.RegisterWithMeta(ctx, req, RequestMeta{})
}

func (s *AuthService) RegisterWithMeta(ctx context.Context, req dto.RegisterRequest, meta RequestMeta) (*dto.TokenPair, *domain.User, error) {
	email := normalizeEmail(req.Email)
	if err := s.validatePassword(ctx, req.Password); err != nil {
		return nil, nil, err
	}
	existing, err := s.userRepo.GetByEmail(ctx, email)
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
		Email:            email,
		PasswordHash:     string(hash),
		DisplayName:      req.DisplayName,
		Timezone:         "UTC",
		DesiredRetention: 0.9,
		DailyNewLimit:    20,
		DailyReviewLimit: 200,
	}

	var tokens *dto.TokenPair
	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}
		if err := s.issueVerificationCode(ctx, user); err != nil {
			return err
		}
		t, err := s.issueTokenPair(ctx, user.ID, meta)
		if err != nil {
			return err
		}
		tokens = t
		return nil
	}); err != nil {
		return nil, nil, err
	}

	s.emitAudit(ctx, domain.AuditEventRegister, &user.ID, meta, map[string]any{"email": email})
	return tokens, user, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenPair, *domain.User, error) {
	return s.LoginWithMeta(ctx, req, RequestMeta{})
}

func (s *AuthService) LoginWithMeta(ctx context.Context, req dto.LoginRequest, meta RequestMeta) (*dto.TokenPair, *domain.User, error) {
	email := normalizeEmail(req.Email)

	if s.policy.LoginLockoutMaxFails > 0 {
		since := s.now().Add(-s.policy.LoginLockoutWindow)
		failures, err := s.laRepo.CountRecentFailures(ctx, email, since)
		if err != nil {
			return nil, nil, err
		}
		if failures >= s.policy.LoginLockoutMaxFails {
			s.emitAudit(ctx, domain.AuditEventLoginFailed, nil, meta, map[string]any{"email": email, "reason": "locked"})
			return nil, nil, domain.ErrAccountLocked
		}
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.recordLogin(ctx, email, meta, false)
			s.emitAudit(ctx, domain.AuditEventLoginFailed, nil, meta, map[string]any{"email": email, "reason": "user_not_found"})
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if user.PasswordHash == "" {
		// OAuth-only account — no password login possible
		s.recordLogin(ctx, email, meta, false)
		s.emitAudit(ctx, domain.AuditEventLoginFailed, &user.ID, meta, map[string]any{"email": email, "reason": "oauth_only"})
		return nil, nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.recordLogin(ctx, email, meta, false)
		s.emitAudit(ctx, domain.AuditEventLoginFailed, &user.ID, meta, map[string]any{"email": email, "reason": "bad_password"})
		return nil, nil, domain.ErrInvalidCredentials
	}

	if s.policy.RequireEmailVerified && !user.IsEmailVerified() {
		s.recordLogin(ctx, email, meta, false)
		s.emitAudit(ctx, domain.AuditEventLoginFailed, &user.ID, meta, map[string]any{"email": email, "reason": "email_unverified"})
		return nil, nil, domain.ErrEmailNotVerified
	}

	newDevice := s.isNewDevice(ctx, user.ID, meta)

	tokens, err := s.issueTokenPair(ctx, user.ID, meta)
	if err != nil {
		return nil, nil, err
	}

	s.recordLogin(ctx, email, meta, true)
	s.emitAudit(ctx, domain.AuditEventLoginSuccess, &user.ID, meta, map[string]any{"email": email, "new_device": newDevice})
	if newDevice && meta.IPAddress != "" {
		s.sendNewDeviceEmail(ctx, user, meta)
	}
	return tokens, user, nil
}

func (s *AuthService) isNewDevice(ctx context.Context, userID uuid.UUID, meta RequestMeta) bool {
	if meta.UserAgent == "" && meta.IPAddress == "" {
		return false // no info to base a signal on
	}
	seen, err := s.rtRepo.DeviceSeen(ctx, userID, meta.UserAgent, meta.IPAddress)
	if err != nil {
		slog.WarnContext(ctx, "failed to check device history", slog.String("error", err.Error()))
		return false
	}
	return !seen
}

func (s *AuthService) recordLogin(ctx context.Context, email string, meta RequestMeta, successful bool) {
	if err := s.laRepo.Create(ctx, &domain.LoginAttempt{
		Email:      email,
		IPAddress:  meta.IPAddress,
		Successful: successful,
	}); err != nil {
		slog.WarnContext(ctx, "failed to record login attempt", slog.String("error", err.Error()))
	}
}

// ── Refresh & logout ──────────────────────────────────────────

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenPair, error) {
	return s.RefreshTokenWithMeta(ctx, refreshToken, RequestMeta{})
}

func (s *AuthService) RefreshTokenWithMeta(ctx context.Context, refreshToken string, meta RequestMeta) (*dto.TokenPair, error) {
	tokenHash := hashToken(refreshToken)
	stored, err := s.rtRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrRefreshTokenInvalid
		}
		return nil, err
	}

	now := s.now()

	// Reuse detection: a previously revoked token is being presented again.
	if stored.RevokedAt != nil {
		// Nuke the entire family.
		if err := s.rtRepo.RevokeAllForUser(ctx, stored.UserID, now); err != nil {
			slog.ErrorContext(ctx, "failed to revoke token family", slog.String("error", err.Error()))
		}
		uid := stored.UserID
		s.emitAudit(ctx, domain.AuditEventRefreshReuseDetected, &uid, meta, nil)
		return nil, domain.ErrRefreshTokenReuse
	}

	if now.After(stored.ExpiresAt) {
		return nil, domain.ErrRefreshTokenInvalid
	}

	// Verify user still exists.
	if _, err := s.userRepo.GetByID(ctx, stored.UserID); err != nil {
		return nil, domain.ErrRefreshTokenInvalid
	}

	var tokens *dto.TokenPair
	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.rtRepo.Revoke(ctx, stored.ID, now); err != nil {
			return err
		}
		t, err := s.issueTokenPairChild(ctx, stored.UserID, meta, &stored.ID)
		if err != nil {
			return err
		}
		tokens = t
		return nil
	}); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	stored, err := s.rtRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil // idempotent
		}
		return err
	}
	if err := s.rtRepo.Revoke(ctx, stored.ID, s.now()); err != nil {
		return err
	}
	uid := stored.UserID
	s.emitAudit(ctx, domain.AuditEventLogout, &uid, RequestMeta{}, nil)
	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if err := s.rtRepo.RevokeAllForUser(ctx, userID, s.now()); err != nil {
		return err
	}
	s.emitAudit(ctx, domain.AuditEventLogoutAll, &userID, RequestMeta{}, nil)
	return nil
}

// ── Email verification ────────────────────────────────────────

func (s *AuthService) issueVerificationCode(ctx context.Context, user *domain.User) error {
	// Remove any existing un-consumed codes first so only one is active.
	if err := s.evRepo.DeleteByUserID(ctx, user.ID); err != nil {
		return err
	}
	code := randomNumericCode(s.policy.OTPLength)
	ev := &domain.EmailVerification{
		UserID:    user.ID,
		CodeHash:  hashToken(code),
		ExpiresAt: s.now().Add(s.policy.OTPTTL),
	}
	if err := s.evRepo.Create(ctx, ev); err != nil {
		return err
	}

	msg := mailer.VerificationEmail(user.DisplayName, code, int(s.policy.OTPTTL.Minutes()))
	msg.To = user.Email
	if err := s.mailer.Send(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "failed to send verification email",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()),
		)
		// Do not fail the transaction on mailer error; user can request resend.
	}
	return nil
}

func (s *AuthService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil // do not disclose existence
		}
		return err
	}
	if user.IsEmailVerified() {
		return domain.ErrEmailAlreadyVerified
	}
	return s.issueVerificationCode(ctx, user)
}

func (s *AuthService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrOTPInvalid
		}
		return err
	}
	if user.IsEmailVerified() {
		return domain.ErrEmailAlreadyVerified
	}

	ev, err := s.evRepo.GetActiveByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrOTPInvalid
		}
		return err
	}

	now := s.now()
	if now.After(ev.ExpiresAt) {
		return domain.ErrOTPInvalid
	}
	if ev.Attempts >= s.policy.OTPMaxAttempts {
		return domain.ErrOTPTooManyAttempts
	}

	if !hmac.Equal([]byte(ev.CodeHash), []byte(hashToken(req.Code))) {
		if err := s.evRepo.IncrementAttempts(ctx, ev.ID); err != nil {
			slog.WarnContext(ctx, "failed to increment otp attempts", slog.String("error", err.Error()))
		}
		return domain.ErrOTPInvalid
	}

	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.evRepo.Consume(ctx, ev.ID, now); err != nil {
			return err
		}
		return s.userRepo.MarkEmailVerified(ctx, user.ID, now)
	}); err != nil {
		return err
	}
	uid := user.ID
	s.emitAudit(ctx, domain.AuditEventEmailVerified, &uid, RequestMeta{}, nil)
	return nil
}

// ── Forgot / reset password ───────────────────────────────────

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil // do not disclose existence
		}
		return err
	}

	if err := s.prtRepo.DeleteByUserID(ctx, user.ID); err != nil {
		return err
	}

	raw, err := randomURLToken(32)
	if err != nil {
		return err
	}
	prt := &domain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashToken(raw),
		ExpiresAt: s.now().Add(s.policy.ResetTokenTTL),
	}
	if err := s.prtRepo.Create(ctx, prt); err != nil {
		return err
	}

	resetURL := buildResetURL(s.frontendURL, raw)
	msg := mailer.PasswordResetEmail(user.DisplayName, resetURL, int(s.policy.ResetTokenTTL.Minutes()))
	msg.To = user.Email
	if err := s.mailer.Send(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "failed to send password reset email",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}
	uid := user.ID
	s.emitAudit(ctx, domain.AuditEventPasswordResetRequested, &uid, RequestMeta{}, nil)
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	if err := s.validatePassword(ctx, req.NewPassword); err != nil {
		return err
	}
	prt, err := s.prtRepo.GetByHash(ctx, hashToken(req.Token))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrResetTokenInvalid
		}
		return err
	}
	now := s.now()
	if prt.ConsumedAt != nil || now.After(prt.ExpiresAt) {
		return domain.ErrResetTokenInvalid
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.userRepo.UpdatePassword(ctx, prt.UserID, string(newHash)); err != nil {
			return err
		}
		if err := s.prtRepo.Consume(ctx, prt.ID, now); err != nil {
			return err
		}
		// Revoke all active sessions.
		return s.rtRepo.RevokeAllForUser(ctx, prt.UserID, now)
	}); err != nil {
		return err
	}
	uid := prt.UserID
	s.emitAudit(ctx, domain.AuditEventPasswordResetCompleted, &uid, RequestMeta{}, nil)
	return nil
}

// ── Change password (authenticated) ───────────────────────────

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.PasswordHash == "" {
		return domain.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}
	if req.OldPassword == req.NewPassword {
		return domain.ErrPasswordSameAsOld
	}
	if err := s.validatePassword(ctx, req.NewPassword); err != nil {
		return err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.userRepo.UpdatePassword(ctx, userID, string(newHash)); err != nil {
			return err
		}
		return s.rtRepo.RevokeAllForUser(ctx, userID, s.now())
	}); err != nil {
		return err
	}
	s.emitAudit(ctx, domain.AuditEventPasswordChanged, &userID, RequestMeta{}, nil)
	s.sendPasswordChangedEmail(ctx, user)
	return nil
}

func (s *AuthService) sendPasswordChangedEmail(ctx context.Context, user *domain.User) {
	msg := mailer.PasswordChangedEmail(user.DisplayName, s.now().UTC().Format("02 Jan 2006 15:04 MST"))
	msg.To = user.Email
	if err := s.mailer.Send(ctx, msg); err != nil {
		slog.WarnContext(ctx, "failed to send password-changed email",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()),
		)
	}
}

func (s *AuthService) sendNewDeviceEmail(ctx context.Context, user *domain.User, meta RequestMeta) {
	revokeURL := strings.TrimRight(s.frontendURL, "/") + "/security/sessions"
	msg := mailer.NewDeviceLoginEmail(
		user.DisplayName, meta.UserAgent, meta.IPAddress,
		s.now().UTC().Format("02 Jan 2006 15:04 MST"), revokeURL,
	)
	msg.To = user.Email
	if err := s.mailer.Send(ctx, msg); err != nil {
		slog.WarnContext(ctx, "failed to send new-device email",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()),
		)
	}
}

// ── Sessions (device management) ──────────────────────────────

func (s *AuthService) ListSessions(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	return s.rtRepo.ListActiveByUser(ctx, userID, s.now())
}

func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	session, err := s.rtRepo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrSessionNotFound
		}
		return err
	}
	if session.UserID != userID {
		return domain.ErrForbidden
	}
	if session.RevokedAt != nil {
		return nil // idempotent
	}
	return s.rtRepo.Revoke(ctx, sessionID, s.now())
}

// ── Account deletion (GDPR / app store) ───────────────────────

func (s *AuthService) RequestAccountDeletion(ctx context.Context, userID uuid.UUID, meta RequestMeta) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsPendingDeletion() {
		return nil, domain.ErrDeletionPending
	}

	now := s.now()
	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.userRepo.RequestDeletion(ctx, userID, now); err != nil {
			return err
		}
		return s.rtRepo.RevokeAllForUser(ctx, userID, now)
	}); err != nil {
		return nil, err
	}
	user.DeletionRequestedAt = &now

	grace := s.policy.AccountDeletionGracePeriod
	metadata := map[string]any{
		"grace_period_hours": int(grace.Hours()),
		"scheduled_deletion": now.Add(grace).UTC().Format(time.RFC3339),
	}
	s.emitAudit(ctx, domain.AuditEventAccountDeletionRequested, &userID, meta, metadata)
	s.sendDeletionRequestedEmail(ctx, user, now, grace)
	return user, nil
}

func (s *AuthService) CancelAccountDeletion(ctx context.Context, userID uuid.UUID, meta RequestMeta) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsPendingDeletion() {
		return nil, domain.ErrDeletionNotPending
	}
	if err := s.userRepo.CancelDeletion(ctx, userID); err != nil {
		return nil, err
	}
	user.DeletionRequestedAt = nil
	s.emitAudit(ctx, domain.AuditEventAccountDeletionCancelled, &userID, meta, nil)
	s.sendDeletionCancelledEmail(ctx, user)
	return user, nil
}

func (s *AuthService) ExportAccountData(ctx context.Context, userID uuid.UUID, meta RequestMeta) (map[string]any, error) {
	if s.exportRepo == nil {
		return nil, domain.ErrInvalidInput
	}
	data, err := s.exportRepo.Dump(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.emitAudit(ctx, domain.AuditEventDataExported, &userID, meta, nil)
	return data, nil
}

// Interface-friendly wrappers used by port.AuthServicer so the handler can call
// through the interface even when it doesn't have RequestMeta (it also has a
// type-assertion fallback for the meta-aware path).
func (s *AuthService) RequestAccountDeletionBasic(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.RequestAccountDeletion(ctx, userID, RequestMeta{})
}

func (s *AuthService) CancelAccountDeletionBasic(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.CancelAccountDeletion(ctx, userID, RequestMeta{})
}

func (s *AuthService) ExportAccountDataBasic(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	return s.ExportAccountData(ctx, userID, RequestMeta{})
}

// PurgeExpiredAccounts hard-deletes users whose grace period has elapsed.
// Safe to run on a schedule. Caller provides batch size (default 100).
func (s *AuthService) PurgeExpiredAccounts(ctx context.Context, batch int) (int, error) {
	if s.policy.AccountDeletionGracePeriod <= 0 {
		return 0, nil
	}
	cutoff := s.now().Add(-s.policy.AccountDeletionGracePeriod)
	ids, err := s.userRepo.ListExpiredDeletions(ctx, cutoff, batch)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := s.userRepo.Delete(ctx, id); err != nil {
			slog.ErrorContext(ctx, "failed to hard-delete user",
				slog.String("user_id", id.String()),
				slog.String("error", err.Error()),
			)
			continue
		}
		uid := id
		s.emitAudit(ctx, domain.AuditEventAccountDeleted, &uid, RequestMeta{}, nil)
	}
	return len(ids), nil
}

func (s *AuthService) sendDeletionRequestedEmail(ctx context.Context, user *domain.User, at time.Time, grace time.Duration) {
	cancelURL := strings.TrimRight(s.frontendURL, "/") + "/account/cancel-deletion"
	days := int(grace.Hours()) / 24
	graceStr := fmt.Sprintf("%d hari", days)
	msg := mailer.AccountDeletionRequestedEmail(
		user.DisplayName,
		at.UTC().Format("02 Jan 2006 15:04 MST"),
		graceStr,
		cancelURL,
	)
	msg.To = user.Email
	if err := s.mailer.Send(ctx, msg); err != nil {
		slog.WarnContext(ctx, "failed to send deletion-requested email",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()),
		)
	}
}

func (s *AuthService) sendDeletionCancelledEmail(ctx context.Context, user *domain.User) {
	msg := mailer.AccountDeletionCancelledEmail(user.DisplayName, s.now().UTC().Format("02 Jan 2006 15:04 MST"))
	msg.To = user.Email
	if err := s.mailer.Send(ctx, msg); err != nil {
		slog.WarnContext(ctx, "failed to send deletion-cancelled email",
			slog.String("user_id", user.ID.String()),
			slog.String("error", err.Error()),
		)
	}
}

// ── Profile ───────────────────────────────────────────────────

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

// ── OAuth ─────────────────────────────────────────────────────

func (s *AuthService) FindOrCreateOAuthUser(ctx context.Context, provider, providerID, email, displayName string, avatarURL *string) (*dto.TokenPair, *domain.User, error) {
	return s.FindOrCreateOAuthUserWithMeta(ctx, provider, providerID, email, displayName, avatarURL, RequestMeta{})
}

func (s *AuthService) FindOrCreateOAuthUserWithMeta(ctx context.Context, provider, providerID, email, displayName string, avatarURL *string, meta RequestMeta) (*dto.TokenPair, *domain.User, error) {
	email = normalizeEmail(email)

	oauthAccount, err := s.userRepo.GetOAuthAccount(ctx, provider, providerID)
	if err == nil {
		user, err := s.userRepo.GetByID(ctx, oauthAccount.UserID)
		if err != nil {
			return nil, nil, err
		}
		tokens, err := s.issueTokenPair(ctx, user.ID, meta)
		if err != nil {
			return nil, nil, err
		}
		uid := user.ID
		s.emitAudit(ctx, domain.AuditEventOAuthLogin, &uid, meta, map[string]any{"provider": provider})
		return tokens, user, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, err
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
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

	var tokens *dto.TokenPair
	now := s.now()
	if err := s.uow.Do(ctx, func(ctx context.Context) error {
		if user.ID == uuid.Nil {
			if err := s.userRepo.Create(ctx, user); err != nil {
				return err
			}
		}
		// OAuth providers return verified email → mark verified if not yet.
		if !user.IsEmailVerified() {
			if err := s.userRepo.MarkEmailVerified(ctx, user.ID, now); err != nil {
				return err
			}
			user.EmailVerifiedAt = &now
		}
		if err := s.userRepo.CreateOAuthAccount(ctx, &domain.OAuthAccount{
			UserID:     user.ID,
			Provider:   provider,
			ProviderID: providerID,
			Email:      email,
			AvatarURL:  avatarURL,
		}); err != nil {
			return err
		}
		t, err := s.issueTokenPair(ctx, user.ID, meta)
		if err != nil {
			return err
		}
		tokens = t
		return nil
	}); err != nil {
		return nil, nil, err
	}

	uid := user.ID
	s.emitAudit(ctx, domain.AuditEventOAuthLinked, &uid, meta, map[string]any{"provider": provider})
	return tokens, user, nil
}

// ── Token helpers ─────────────────────────────────────────────

func (s *AuthService) issueTokenPair(ctx context.Context, userID uuid.UUID, meta RequestMeta) (*dto.TokenPair, error) {
	return s.issueTokenPairChild(ctx, userID, meta, nil)
}

func (s *AuthService) issueTokenPairChild(ctx context.Context, userID uuid.UUID, meta RequestMeta, parentID *uuid.UUID) (*dto.TokenPair, error) {
	now := s.now()

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

	raw, err := randomURLToken(32)
	if err != nil {
		return nil, err
	}
	rt := &domain.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ParentID:  parentID,
		UserAgent: meta.UserAgent,
		IPAddress: meta.IPAddress,
		ExpiresAt: now.Add(s.refreshDuration),
	}
	if err := s.rtRepo.Create(ctx, rt); err != nil {
		return nil, err
	}

	return &dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: raw,
		ExpiresAt:    now.Add(s.accessDuration).Unix(),
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomNumericCode(length int) string {
	if length <= 0 {
		length = 6
	}
	const digits = "0123456789"
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			// fallback: time-seeded pseudo (extremely unlikely path)
			b[0] = byte(time.Now().UnixNano())
		}
		buf[i] = digits[int(b[0])%len(digits)]
	}
	return string(buf)
}

func randomURLToken(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func buildResetURL(frontendURL, token string) string {
	u := strings.TrimRight(frontendURL, "/")
	return fmt.Sprintf("%s/reset-password?token=%s", u, url.QueryEscape(token))
}
