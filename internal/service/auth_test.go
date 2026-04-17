package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/alfariesh/backend-quiz/internal/domain"
	"github.com/alfariesh/backend-quiz/internal/dto"
	mockdomain "github.com/alfariesh/backend-quiz/internal/mocks/domain"
	"github.com/alfariesh/backend-quiz/pkg/mailer"
	"github.com/alfariesh/backend-quiz/pkg/password"
)

const testJWTSecret = "test-secret-key-for-testing"

// ── Test doubles for new repos (simple in-memory) ────────────

type fakeEVRepo struct {
	mu      sync.Mutex
	byUser  map[uuid.UUID]*domain.EmailVerification
	counter int
}

func newFakeEVRepo() *fakeEVRepo {
	return &fakeEVRepo{byUser: map[uuid.UUID]*domain.EmailVerification{}}
}
func (r *fakeEVRepo) Create(ctx context.Context, ev *domain.EmailVerification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counter++
	ev.ID = uuid.New()
	ev.CreatedAt = time.Now()
	r.byUser[ev.UserID] = ev
	return nil
}
func (r *fakeEVRepo) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailVerification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ev, ok := r.byUser[userID]
	if !ok || ev.ConsumedAt != nil {
		return nil, domain.ErrNotFound
	}
	return ev, nil
}
func (r *fakeEVRepo) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ev := range r.byUser {
		if ev.ID == id {
			ev.Attempts++
			return nil
		}
	}
	return nil
}
func (r *fakeEVRepo) Consume(ctx context.Context, id uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ev := range r.byUser {
		if ev.ID == id {
			ev.ConsumedAt = &at
			return nil
		}
	}
	return nil
}
func (r *fakeEVRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byUser, userID)
	return nil
}

type fakePRTRepo struct {
	mu      sync.Mutex
	byHash  map[string]*domain.PasswordResetToken
}

func newFakePRTRepo() *fakePRTRepo {
	return &fakePRTRepo{byHash: map[string]*domain.PasswordResetToken{}}
}
func (r *fakePRTRepo) Create(ctx context.Context, t *domain.PasswordResetToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	r.byHash[t.TokenHash] = t
	return nil
}
func (r *fakePRTRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byHash[tokenHash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (r *fakePRTRepo) Consume(ctx context.Context, id uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.byHash {
		if t.ID == id {
			t.ConsumedAt = &at
			return nil
		}
	}
	return nil
}
func (r *fakePRTRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, t := range r.byHash {
		if t.UserID == userID {
			delete(r.byHash, hash)
		}
	}
	return nil
}

type fakeRTRepo struct {
	mu     sync.Mutex
	byHash map[string]*domain.RefreshToken
	byID   map[uuid.UUID]*domain.RefreshToken
}

func newFakeRTRepo() *fakeRTRepo {
	return &fakeRTRepo{
		byHash: map[string]*domain.RefreshToken{},
		byID:   map[uuid.UUID]*domain.RefreshToken{},
	}
}
func (r *fakeRTRepo) Create(ctx context.Context, t *domain.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	r.byHash[t.TokenHash] = t
	r.byID[t.ID] = t
	return nil
}
func (r *fakeRTRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byHash[tokenHash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (r *fakeRTRepo) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.byID[id]; ok && t.RevokedAt == nil {
		t.RevokedAt = &at
	}
	return nil
}
func (r *fakeRTRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.byID {
		if t.UserID == userID && t.RevokedAt == nil {
			revokedAt := at
			t.RevokedAt = &revokedAt
		}
	}
	return nil
}
func (r *fakeRTRepo) RevokeChildren(ctx context.Context, parentID uuid.UUID, at time.Time) error {
	return nil
}
func (r *fakeRTRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (r *fakeRTRepo) ListActiveByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]*domain.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.RefreshToken
	for _, t := range r.byID {
		if t.UserID == userID && t.RevokedAt == nil && t.ExpiresAt.After(now) {
			out = append(out, t)
		}
	}
	return out, nil
}
func (r *fakeRTRepo) DeviceSeen(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.byID {
		if t.UserID == userID && t.UserAgent == userAgent && t.IPAddress == ipAddress {
			return true, nil
		}
	}
	return false, nil
}

type fakeLARepo struct {
	mu       sync.Mutex
	attempts []domain.LoginAttempt
}

func newFakeLARepo() *fakeLARepo { return &fakeLARepo{} }
func (r *fakeLARepo) Create(ctx context.Context, a *domain.LoginAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	r.attempts = append(r.attempts, *a)
	return nil
}
func (r *fakeLARepo) CountRecentFailures(ctx context.Context, email string, since time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, a := range r.attempts {
		if a.Email == email && !a.Successful && !a.CreatedAt.Before(since) {
			n++
		}
	}
	return n, nil
}

type capturingMailer struct {
	sent []mailer.Message
}

func (m *capturingMailer) Send(ctx context.Context, msg mailer.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

type fakeAuditRepo struct {
	mu   sync.Mutex
	logs []domain.AuthAuditLog
}

func newFakeAuditRepo() *fakeAuditRepo { return &fakeAuditRepo{} }
func (r *fakeAuditRepo) Create(ctx context.Context, l *domain.AuthAuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l.ID = uuid.New()
	l.CreatedAt = time.Now()
	r.logs = append(r.logs, *l)
	return nil
}

type fakeExportRepo struct {
	data map[string]any
	err  error
}

func (r *fakeExportRepo) Dump(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.data != nil {
		return r.data, nil
	}
	return map[string]any{"user": map[string]any{"id": userID.String()}}, nil
}

type testDeps struct {
	userRepo   *mockdomain.MockUserRepository
	evRepo     *fakeEVRepo
	prtRepo    *fakePRTRepo
	rtRepo     *fakeRTRepo
	laRepo     *fakeLARepo
	auditRepo  *fakeAuditRepo
	exportRepo *fakeExportRepo
	mailer     *capturingMailer
	svc        *AuthService
}

func defaultPolicy() AuthPolicy {
	return AuthPolicy{
		OTPLength:            6,
		OTPTTL:               15 * time.Minute,
		OTPMaxAttempts:       5,
		ResetTokenTTL:        15 * time.Minute,
		LoginLockoutWindow:   15 * time.Minute,
		LoginLockoutMaxFails: 5,
		RequireEmailVerified: false,
	}
}

func newTestSvc(t *testing.T) testDeps {
	t.Helper()
	userRepo := mockdomain.NewMockUserRepository(t)
	ev := newFakeEVRepo()
	prt := newFakePRTRepo()
	rt := newFakeRTRepo()
	la := newFakeLARepo()
	audit := newFakeAuditRepo()
	export := &fakeExportRepo{}
	m := &capturingMailer{}

	svc := NewAuthService(
		userRepo, ev, prt, rt, la, audit, export,
		noopUoW{}, m,
		password.Policy{MinLength: 8, MaxLength: 72},
		testJWTSecret, 15*time.Minute, 720*time.Hour,
		"http://localhost:3000",
		defaultPolicy(),
	)
	return testDeps{
		userRepo:   userRepo,
		evRepo:     ev,
		prtRepo:    prt,
		rtRepo:     rt,
		laRepo:     la,
		auditRepo:  audit,
		exportRepo: export,
		mailer:     m,
		svc:        svc,
	}
}

// ── Register ────────────────────────────────────────────────

func TestAuthService_Register_Success(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(nil, domain.ErrNotFound)
	d.userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	tokens, user, err := d.svc.Register(ctx, dto.RegisterRequest{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test User",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Len(t, d.mailer.sent, 1, "verification email should be sent")
}

func TestAuthService_Register_EmailTaken(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	existing := &domain.User{ID: uuid.New(), Email: "test@example.com"}
	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(existing, nil)

	_, _, err := d.svc.Register(ctx, dto.RegisterRequest{
		Email: "test@example.com", Password: "password123", DisplayName: "Test",
	})
	assert.ErrorIs(t, err, domain.ErrEmailTaken)
}

// ── Login ──────────────────────────────────────────────────

func TestAuthService_Login_Success(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Email: "test@example.com", PasswordHash: string(hash)}
	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(user, nil)

	tokens, u, err := d.svc.Login(ctx, dto.LoginRequest{Email: "test@example.com", Password: "password123"})
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, user.ID, u.ID)
	assert.Equal(t, 1, len(d.laRepo.attempts))
	assert.True(t, d.laRepo.attempts[0].Successful)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Email: "test@example.com", PasswordHash: string(hash)}
	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(user, nil)

	_, _, err := d.svc.Login(ctx, dto.LoginRequest{Email: "test@example.com", Password: "wrong"})
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.False(t, d.laRepo.attempts[0].Successful)
}

func TestAuthService_Login_Lockout(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	// Seed 5 recent failures.
	for i := 0; i < 5; i++ {
		_ = d.laRepo.Create(ctx, &domain.LoginAttempt{Email: "test@example.com", Successful: false})
	}

	_, _, err := d.svc.Login(ctx, dto.LoginRequest{Email: "test@example.com", Password: "anything"})
	assert.ErrorIs(t, err, domain.ErrAccountLocked)
}

// ── Refresh token ──────────────────────────────────────────

func TestAuthService_RefreshToken_Success(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(nil, domain.ErrNotFound)
	d.userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(func(_ context.Context, u *domain.User) error {
		u.ID = userID
		return nil
	})
	d.userRepo.On("GetByID", ctx, userID).Return(&domain.User{ID: userID}, nil)

	tokens, _, err := d.svc.Register(ctx, dto.RegisterRequest{
		Email: "test@example.com", Password: "password123", DisplayName: "T",
	})
	require.NoError(t, err)

	newTokens, err := d.svc.RefreshToken(ctx, tokens.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEqual(t, tokens.RefreshToken, newTokens.RefreshToken, "refresh token should rotate")
}

func TestAuthService_RefreshToken_ReuseDetected(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(nil, domain.ErrNotFound)
	d.userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(func(_ context.Context, u *domain.User) error {
		u.ID = userID
		return nil
	})
	d.userRepo.On("GetByID", ctx, userID).Return(&domain.User{ID: userID}, nil)

	tokens, _, err := d.svc.Register(ctx, dto.RegisterRequest{
		Email: "test@example.com", Password: "password123", DisplayName: "T",
	})
	require.NoError(t, err)

	// First refresh — succeeds.
	_, err = d.svc.RefreshToken(ctx, tokens.RefreshToken)
	require.NoError(t, err)

	// Reusing the original (now revoked) refresh token — should be detected.
	_, err = d.svc.RefreshToken(ctx, tokens.RefreshToken)
	assert.ErrorIs(t, err, domain.ErrRefreshTokenReuse)
}

func TestAuthService_RefreshToken_Invalid(t *testing.T) {
	d := newTestSvc(t)
	_, err := d.svc.RefreshToken(context.Background(), "not-a-real-token")
	assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
}

// ── Email verification ─────────────────────────────────────

func TestAuthService_VerifyEmail_Success(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{ID: userID, Email: "test@example.com", DisplayName: "T"}
	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(user, nil)
	d.userRepo.On("MarkEmailVerified", ctx, userID, mock.Anything).Return(nil)

	// Manually seed a code so we know what to submit.
	require.NoError(t, d.svc.issueVerificationCode(ctx, user))
	require.Len(t, d.mailer.sent, 1)
	sentBody := d.mailer.sent[0].TextBody
	// Extract code from body (6 digits).
	var code string
	for i := 0; i < len(sentBody)-5; i++ {
		if isAllDigits(sentBody[i : i+6]) {
			code = sentBody[i : i+6]
			break
		}
	}
	require.NotEmpty(t, code, "should find 6-digit code in email body")

	err := d.svc.VerifyEmail(ctx, dto.VerifyEmailRequest{Email: "test@example.com", Code: code})
	assert.NoError(t, err)
}

func TestAuthService_VerifyEmail_WrongCode(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{ID: userID, Email: "test@example.com", DisplayName: "T"}
	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(user, nil)

	require.NoError(t, d.svc.issueVerificationCode(ctx, user))

	err := d.svc.VerifyEmail(ctx, dto.VerifyEmailRequest{Email: "test@example.com", Code: "000000"})
	assert.ErrorIs(t, err, domain.ErrOTPInvalid)
}

// ── Password reset ─────────────────────────────────────────

func TestAuthService_ForgotResetPassword_Flow(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	user := &domain.User{ID: userID, Email: "test@example.com", PasswordHash: string(hash), DisplayName: "T"}
	d.userRepo.On("GetByEmail", ctx, "test@example.com").Return(user, nil)

	require.NoError(t, d.svc.ForgotPassword(ctx, "test@example.com"))
	require.Len(t, d.mailer.sent, 1, "reset email sent")

	// Grab raw token from URL (we don't have it; use test hook: inspect stored token → but we only stored hash).
	// Extract token query param from email body.
	body := d.mailer.sent[0].TextBody
	// Find "token=" and extract until newline or end.
	tokenStart := -1
	for i := 0; i+6 < len(body); i++ {
		if body[i:i+6] == "token=" {
			tokenStart = i + 6
			break
		}
	}
	require.GreaterOrEqual(t, tokenStart, 0, "token= found in email")
	tokenEnd := len(body)
	for i := tokenStart; i < len(body); i++ {
		if body[i] == '\n' || body[i] == ' ' {
			tokenEnd = i
			break
		}
	}
	rawToken := body[tokenStart:tokenEnd]
	// URL-decode any %XX.
	rawToken, err := urlDecode(rawToken)
	require.NoError(t, err)

	d.userRepo.On("UpdatePassword", ctx, userID, mock.AnythingOfType("string")).Return(nil)

	require.NoError(t, d.svc.ResetPassword(ctx, dto.ResetPasswordRequest{
		Token: rawToken, NewPassword: "new-password-123",
	}))
}

// ── Change password ────────────────────────────────────────

func TestAuthService_ChangePassword_Success(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	user := &domain.User{ID: userID, Email: "t@e.com", PasswordHash: string(hash)}
	d.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	d.userRepo.On("UpdatePassword", ctx, userID, mock.AnythingOfType("string")).Return(nil)

	err := d.svc.ChangePassword(ctx, userID, dto.ChangePasswordRequest{
		OldPassword: "old-password", NewPassword: "brand-new-pw-99",
	})
	assert.NoError(t, err)
}

func TestAuthService_ChangePassword_WrongOld(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	user := &domain.User{ID: userID, PasswordHash: string(hash)}
	d.userRepo.On("GetByID", ctx, userID).Return(user, nil)

	err := d.svc.ChangePassword(ctx, userID, dto.ChangePasswordRequest{
		OldPassword: "wrong", NewPassword: "brand-new-pw-99",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

// ── Account deletion ───────────────────────────────────────

func TestAuthService_RequestAccountDeletion_Success(t *testing.T) {
	d := newTestSvc(t)
	d.svc.policy.AccountDeletionGracePeriod = 30 * 24 * time.Hour
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{ID: userID, Email: "t@e.com", DisplayName: "Tester"}

	d.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	d.userRepo.On("RequestDeletion", ctx, userID, mock.AnythingOfType("time.Time")).Return(nil)

	got, err := d.svc.RequestAccountDeletion(ctx, userID, RequestMeta{IPAddress: "1.2.3.4"})
	require.NoError(t, err)
	require.NotNil(t, got.DeletionRequestedAt)

	// Email sent + audit emitted
	assert.Len(t, d.mailer.sent, 1)
	assert.Contains(t, d.mailer.sent[0].Subject, "penghapusan")
	require.Len(t, d.auditRepo.logs, 1)
	assert.Equal(t, domain.AuditEventAccountDeletionRequested, d.auditRepo.logs[0].Event)
}

func TestAuthService_RequestAccountDeletion_AlreadyPending(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()
	user := &domain.User{ID: userID, Email: "t@e.com", DeletionRequestedAt: &now}
	d.userRepo.On("GetByID", ctx, userID).Return(user, nil)

	_, err := d.svc.RequestAccountDeletion(ctx, userID, RequestMeta{})
	assert.ErrorIs(t, err, domain.ErrDeletionPending)
	assert.Empty(t, d.mailer.sent)
}

func TestAuthService_CancelAccountDeletion_Success(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()
	user := &domain.User{ID: userID, Email: "t@e.com", DisplayName: "Tester", DeletionRequestedAt: &now}
	d.userRepo.On("GetByID", ctx, userID).Return(user, nil)
	d.userRepo.On("CancelDeletion", ctx, userID).Return(nil)

	got, err := d.svc.CancelAccountDeletion(ctx, userID, RequestMeta{})
	require.NoError(t, err)
	assert.Nil(t, got.DeletionRequestedAt)

	assert.Len(t, d.mailer.sent, 1)
	require.Len(t, d.auditRepo.logs, 1)
	assert.Equal(t, domain.AuditEventAccountDeletionCancelled, d.auditRepo.logs[0].Event)
}

func TestAuthService_CancelAccountDeletion_NotPending(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()

	userID := uuid.New()
	user := &domain.User{ID: userID, Email: "t@e.com"}
	d.userRepo.On("GetByID", ctx, userID).Return(user, nil)

	_, err := d.svc.CancelAccountDeletion(ctx, userID, RequestMeta{})
	assert.ErrorIs(t, err, domain.ErrDeletionNotPending)
}

func TestAuthService_ExportAccountData(t *testing.T) {
	d := newTestSvc(t)
	ctx := context.Background()
	userID := uuid.New()
	d.exportRepo.data = map[string]any{"user": map[string]any{"id": userID.String()}, "decks": []any{}}

	got, err := d.svc.ExportAccountData(ctx, userID, RequestMeta{})
	require.NoError(t, err)
	assert.Contains(t, got, "user")
	require.Len(t, d.auditRepo.logs, 1)
	assert.Equal(t, domain.AuditEventDataExported, d.auditRepo.logs[0].Event)
}

func TestAuthService_PurgeExpiredAccounts(t *testing.T) {
	d := newTestSvc(t)
	d.svc.policy.AccountDeletionGracePeriod = 30 * 24 * time.Hour
	ctx := context.Background()

	id1, id2 := uuid.New(), uuid.New()
	d.userRepo.On("ListExpiredDeletions", ctx, mock.AnythingOfType("time.Time"), 50).Return([]uuid.UUID{id1, id2}, nil)
	d.userRepo.On("Delete", ctx, id1).Return(nil)
	d.userRepo.On("Delete", ctx, id2).Return(nil)

	n, err := d.svc.PurgeExpiredAccounts(ctx, 50)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Len(t, d.auditRepo.logs, 2)
}

// ── Helpers ────────────────────────────────────────────────

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func urlDecode(s string) (string, error) {
	// Minimal percent-decoding for test convenience.
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			high := fromHex(s[i+1])
			low := fromHex(s[i+2])
			if high < 0 || low < 0 {
				out = append(out, s[i])
				continue
			}
			out = append(out, byte(high<<4|low))
			i += 2
		} else {
			out = append(out, s[i])
		}
	}
	return string(out), nil
}

func fromHex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
