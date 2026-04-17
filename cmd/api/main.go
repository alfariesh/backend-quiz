package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"

	"github.com/alfariesh/backend-quiz/config"
	"github.com/alfariesh/backend-quiz/internal/handler"
	"github.com/alfariesh/backend-quiz/internal/middleware"
	"github.com/alfariesh/backend-quiz/internal/repository"
	"github.com/alfariesh/backend-quiz/internal/service"
	"github.com/alfariesh/backend-quiz/pkg/captcha"
	"github.com/alfariesh/backend-quiz/pkg/mailer"
	"github.com/alfariesh/backend-quiz/pkg/oauth"
	"github.com/alfariesh/backend-quiz/pkg/password"
	"github.com/alfariesh/backend-quiz/pkg/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Setup logger
	zl := zerolog.New(os.Stdout).With().Timestamp().Logger()
	if cfg.Server.Environment == "development" {
		zl = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	}
	logger := slog.New(slogzerolog.Option{Logger: &zl}.NewZerologHandler())
	slog.SetDefault(logger)

	// Database pool
	ctx := context.Background()
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("parsing database URL: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.Database.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}
	logger.Info("connected to database")

	// Repositories
	userRepo := repository.NewUserRepository(pool)
	deckRepo := repository.NewDeckRepository(pool)
	cardRepo := repository.NewCardRepository(pool)
	reviewRepo := repository.NewReviewRepository(pool)
	sessionRepo := repository.NewStudySessionRepository(pool)
	statsRepo := repository.NewStatsRepository(pool)
	quizRepo := repository.NewQuizRepository(pool)
	quizAttemptRepo := repository.NewQuizAttemptRepository(pool)
	mediaRepo := repository.NewMediaRepository(pool)
	goalRepo := repository.NewGoalRepository(pool)
	evRepo := repository.NewEmailVerificationRepository(pool)
	prtRepo := repository.NewPasswordResetTokenRepository(pool)
	rtRepo := repository.NewRefreshTokenRepository(pool)
	laRepo := repository.NewLoginAttemptRepository(pool)
	auditRepo := repository.NewAuthAuditLogRepository(pool)
	exportRepo := repository.NewUserDataExportRepository(pool)

	// Unit of Work
	uow := repository.NewUnitOfWork(pool)

	// Mailer
	var mailClient mailer.Mailer
	if cfg.Mailer.ResendAPIKey != "" {
		mailClient = mailer.NewResendClient(cfg.Mailer.ResendAPIKey, cfg.Mailer.FromAddress, cfg.Mailer.ReplyTo)
		logger.Info("mailer: Resend configured")
	} else {
		mailClient = mailer.NoopMailer{}
		logger.Warn("mailer: RESEND_API_KEY not set — emails will not be sent")
	}

	// Google OAuth
	var googleClient *oauth.GoogleClient
	if cfg.GoogleOAuth.Enabled() {
		googleClient = oauth.NewGoogleClient(cfg.GoogleOAuth.ClientID, cfg.GoogleOAuth.ClientSecret, cfg.GoogleOAuth.RedirectURL)
		logger.Info("google oauth configured")
	} else {
		logger.Warn("google oauth not configured")
	}

	// Services
	authPolicy := service.AuthPolicy{
		OTPLength:                  cfg.AuthPolicy.OTPLength,
		OTPTTL:                     cfg.AuthPolicy.OTPTTL,
		OTPMaxAttempts:             cfg.AuthPolicy.OTPMaxAttempts,
		ResetTokenTTL:              cfg.AuthPolicy.ResetTokenTTL,
		LoginLockoutWindow:         cfg.AuthPolicy.LoginLockoutWindow,
		LoginLockoutMaxFails:       cfg.AuthPolicy.LoginLockoutMaxFails,
		RequireEmailVerified:       cfg.AuthPolicy.RequireEmailVerified,
		AccountDeletionGracePeriod: cfg.AuthPolicy.DeletionGracePeriod,
	}
	pwPolicy := password.Policy{
		MinLength:    cfg.AuthPolicy.PasswordMinLength,
		MaxLength:    cfg.AuthPolicy.PasswordMaxLength,
		RequireMixed: cfg.AuthPolicy.PasswordRequireMixed,
	}
	if cfg.AuthPolicy.PasswordHIBPCheck {
		pwPolicy.HIBPChecker = password.NewHIBPChecker()
		logger.Info("password policy: HIBP breach check enabled")
	}

	authSvc := service.NewAuthService(
		userRepo, evRepo, prtRepo, rtRepo, laRepo, auditRepo, exportRepo,
		uow, mailClient, pwPolicy,
		cfg.JWT.Secret, cfg.JWT.AccessDuration, cfg.JWT.RefreshDuration,
		cfg.App.FrontendURL,
		authPolicy,
	)
	deckSvc := service.NewDeckService(deckRepo, cardRepo, uow)
	cardSvc := service.NewCardService(cardRepo, deckRepo)
	studySvc := service.NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, uow)
	statsSvc := service.NewStatsService(reviewRepo, sessionRepo, statsRepo, cardRepo, deckRepo, quizRepo, quizAttemptRepo)
	quizSvc := service.NewQuizService(quizRepo, quizAttemptRepo, cardRepo, deckRepo, reviewRepo, uow)

	r2Client := storage.NewR2Client(cfg.R2.AccountID, cfg.R2.AccessKeyID, cfg.R2.SecretAccessKey, cfg.R2.BucketName, cfg.R2.PublicURL)
	mediaSvc := service.NewMediaService(mediaRepo, cardRepo, deckRepo, r2Client, cfg.R2)
	goalSvc := service.NewGoalService(goalRepo, reviewRepo)
	ragSvc := service.NewRAGService(pool, cfg.RAG.ServiceURL, cfg.RAG.InternalToken, cfg.RAG.Timeout, cfg.RAG.DailyLimit)

	// Handlers
	healthH := handler.NewHealthHandler(pool)
	authH := handler.NewAuthHandler(authSvc, handler.AuthHandlerOptions{
		Google:           googleClient,
		SuccessRedirect:  cfg.GoogleOAuth.SuccessRedirect,
		FailureRedirect:  cfg.GoogleOAuth.FailureRedirect,
		SecureCookies:    cfg.Server.Environment == "production",
		TrustForwardedIP: cfg.Security.TrustForwardedFor,
	})
	deckH := handler.NewDeckHandler(deckSvc)
	cardH := handler.NewCardHandler(cardSvc)
	studyH := handler.NewStudyHandler(studySvc)
	statsH := handler.NewStatsHandler(statsSvc)
	quizH := handler.NewQuizHandler(quizSvc)
	mediaH := handler.NewMediaHandler(mediaSvc)
	goalH := handler.NewGoalHandler(goalSvc)
	// SSE passes through the handler's own client (no request timeout; streams can run long).
	ragStreamClient := &http.Client{}
	ragH := handler.NewRAGHandler(ragSvc, cfg.RAG.ServiceURL, cfg.RAG.InternalToken, cfg.RAG.StreamTimeout, ragStreamClient)

	// Router
	r := chi.NewRouter()

	// Captcha verifier
	var captchaVerifier captcha.Verifier
	if cfg.Captcha.Enabled() {
		captchaVerifier = captcha.NewTurnstile(cfg.Captcha.TurnstileSecret)
		logger.Info("captcha: Turnstile configured")
	} else {
		captchaVerifier = captcha.NoopVerifier{}
		logger.Warn("captcha: TURNSTILE_SECRET_KEY not set — captcha check disabled")
	}

	// Global middleware
	rl := middleware.NewRateLimiterWithProxy(10, 20, cfg.Security.TrustForwardedFor)
	authRL := middleware.NewRateLimiterWithProxy(cfg.Security.AuthRateLimitRPS, cfg.Security.AuthRateLimitBurst, cfg.Security.TrustForwardedFor)
	captchaMW := middleware.Captcha(captchaVerifier, cfg.Security.TrustForwardedFor)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.SecurityHeaders(middleware.SecurityHeadersOptions{
		HSTSMaxAgeSeconds:     cfg.Security.HSTSMaxAge,
		HSTSIncludeSubdomains: true,
		ContentSecurityPolicy: cfg.Security.ContentSecurityPolicy,
	}))
	r.Use(middleware.CORS(cfg.Security.AllowedOrigins...))
	r.Use(rl.Middleware)

	// Health endpoints
	r.Get("/healthz", healthH.Healthz)
	r.Get("/readyz", healthH.Readyz)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Route("/auth", func(r chi.Router) {
			// Refresh + logout: stricter IP rate limit only (no captcha — used on normal traffic).
			r.Group(func(r chi.Router) {
				r.Use(authRL.Middleware)
				r.Post("/refresh", authH.Refresh)
				r.Post("/logout", authH.Logout)
				r.Post("/verify-email", authH.VerifyEmail)
			})

			// Abuse-prone endpoints: stricter IP rate limit + captcha.
			r.Group(func(r chi.Router) {
				r.Use(authRL.Middleware)
				r.Use(captchaMW)
				r.Post("/register", authH.Register)
				r.Post("/login", authH.Login)
				r.Post("/resend-verification", authH.ResendVerification)
				r.Post("/forgot-password", authH.ForgotPassword)
				r.Post("/reset-password", authH.ResetPassword)
			})

			// OAuth: stricter rate limit only (captcha not applicable to redirect flow).
			r.Group(func(r chi.Router) {
				r.Use(authRL.Middleware)
				r.Get("/google", authH.GoogleRedirect)
				r.Get("/google/callback", authH.GoogleCallback)
			})
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWT.Secret))

			// User
			r.Get("/auth/me", authH.Me)
			r.Put("/auth/me", authH.UpdateProfile)
			r.Delete("/auth/me", authH.DeleteAccount)
			r.Post("/auth/me/cancel-deletion", authH.CancelAccountDeletion)
			r.Get("/auth/me/export", authH.ExportAccountData)
			r.Post("/auth/change-password", authH.ChangePassword)
			r.Post("/auth/logout-all", authH.LogoutAll)
			r.Get("/auth/sessions", authH.ListSessions)
			r.Delete("/auth/sessions/{sessionID}", authH.RevokeSession)

			// Decks
			r.Route("/decks", func(r chi.Router) {
				r.Get("/", deckH.List)
				r.Post("/", deckH.Create)
				r.Get("/public", deckH.ListPublic)
				r.Post("/import", deckH.Import)
				r.Post("/clone/{shareCode}", deckH.Clone)

				r.Route("/{deckID}", func(r chi.Router) {
					r.Get("/", deckH.Get)
					r.Put("/", deckH.Update)
					r.Delete("/", deckH.Delete)
					r.Get("/export", deckH.Export)
					r.Post("/share", deckH.Share)
					r.Delete("/share", deckH.Unshare)

					// Cards within deck
					r.Route("/cards", func(r chi.Router) {
						r.Get("/", cardH.List)
						r.Post("/", cardH.Create)
						r.Post("/batch", cardH.BatchCreate)
					})
				})
			})

			// Cards (direct access)
			r.Route("/cards/{cardID}", func(r chi.Router) {
				r.Get("/", cardH.Get)
				r.Put("/", cardH.Update)
				r.Delete("/", cardH.Delete)
				r.Put("/suspend", cardH.Suspend)
				r.Put("/reset", cardH.ResetFSRS)

				// Media
				r.Post("/media", mediaH.Upload)
				r.Get("/media", mediaH.ListByCard)
			})

			// Media (direct access)
			r.Delete("/media/{mediaID}", mediaH.Delete)

			// Study
			r.Route("/study", func(r chi.Router) {
				r.Get("/reminders", studyH.Reminders)
				r.Post("/sessions", studyH.StartSession)
				r.Get("/sessions/{sessionID}", studyH.GetSession)
				r.Post("/sessions/{sessionID}/review", studyH.SubmitReview)
				r.Post("/sessions/{sessionID}/reviews/batch", studyH.BatchReview)
				r.Put("/sessions/{sessionID}/end", studyH.EndSession)
			})

			// Stats
			r.Route("/stats", func(r chi.Router) {
				r.Get("/overview", statsH.Overview)
				r.Get("/heatmap", statsH.Heatmap)
				r.Get("/forecast", statsH.Forecast)
				r.Get("/leaderboard", statsH.Leaderboard)
				r.Get("/mastery", statsH.Mastery)
				r.Get("/weak-areas", statsH.WeakAreas)
				r.Get("/test-comparison/{deckID}", statsH.TestComparison)
				r.Get("/deck/{deckID}", statsH.DeckStats)
			})

			// Goals
			r.Route("/goals", func(r chi.Router) {
				r.Post("/", goalH.SetGoal)
				r.Get("/", goalH.ListWithProgress)
				r.Delete("/{goalID}", goalH.DeleteGoal)
			})

			// RAG (proxy → Python rag-service). Per-user rate limit on top of
			// the global IP limit — RAG calls cost LLM tokens, so we keep them
			// tight per authenticated user.
			ragUserRL := middleware.NewUserRateLimiter(cfg.RAG.UserRPS, cfg.RAG.UserBurst)
			r.Route("/rag", func(r chi.Router) {
				r.Use(ragUserRL.Middleware)
				r.Post("/query", ragH.Query)
				r.Post("/query/stream", ragH.QueryStream)
			})

			// Quizzes
			r.Route("/quizzes", func(r chi.Router) {
				r.Get("/", quizH.ListQuizzes)
				r.Post("/", quizH.CreateQuiz)

				r.Route("/{quizID}", func(r chi.Router) {
					r.Get("/", quizH.GetQuiz)
					r.Put("/", quizH.UpdateQuiz)
					r.Delete("/", quizH.DeleteQuiz)

					// Questions
					r.Route("/questions", func(r chi.Router) {
						r.Post("/", quizH.AddQuestion)
						r.Post("/batch", quizH.BatchAddQuestions)
						r.Post("/generate", quizH.GenerateFromDeck)
					r.Post("/generate-ayat", quizH.GenerateAyatQuiz)
						r.Put("/{questionID}", quizH.UpdateQuestion)
						r.Delete("/{questionID}", quizH.DeleteQuestion)
					})

					// Attempts
					r.Route("/attempts", func(r chi.Router) {
						r.Post("/", quizH.StartAttempt)
						r.Get("/", quizH.ListAttempts)
						r.Get("/{attemptID}", quizH.GetAttempt)
						r.Post("/{attemptID}/answer", quizH.SubmitAnswer)
						r.Put("/{attemptID}/complete", quizH.CompleteAttempt)
					})
				})
			})
		})
	})

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", slog.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-done
	logger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
