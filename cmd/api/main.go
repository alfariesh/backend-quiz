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

	// Unit of Work
	uow := repository.NewUnitOfWork(pool)

	// Services
	authSvc := service.NewAuthService(userRepo, uow, cfg.JWT.Secret, cfg.JWT.AccessDuration, cfg.JWT.RefreshDuration)
	deckSvc := service.NewDeckService(deckRepo, cardRepo, uow)
	cardSvc := service.NewCardService(cardRepo, deckRepo)
	studySvc := service.NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, statsRepo, uow)
	statsSvc := service.NewStatsService(reviewRepo, sessionRepo, statsRepo, cardRepo, deckRepo, quizRepo, quizAttemptRepo)
	quizSvc := service.NewQuizService(quizRepo, quizAttemptRepo, cardRepo, deckRepo, reviewRepo, uow)

	r2Client := storage.NewR2Client(cfg.R2.AccountID, cfg.R2.AccessKeyID, cfg.R2.SecretAccessKey, cfg.R2.BucketName, cfg.R2.PublicURL)
	mediaSvc := service.NewMediaService(mediaRepo, cardRepo, deckRepo, r2Client, cfg.R2)
	goalSvc := service.NewGoalService(goalRepo, reviewRepo)

	// Handlers
	healthH := handler.NewHealthHandler(pool)
	authH := handler.NewAuthHandler(authSvc)
	deckH := handler.NewDeckHandler(deckSvc)
	cardH := handler.NewCardHandler(cardSvc)
	studyH := handler.NewStudyHandler(studySvc)
	statsH := handler.NewStatsHandler(statsSvc)
	quizH := handler.NewQuizHandler(quizSvc)
	mediaH := handler.NewMediaHandler(mediaSvc)
	goalH := handler.NewGoalHandler(goalSvc)

	// Router
	r := chi.NewRouter()

	// Global middleware
	rl := middleware.NewRateLimiter(10, 20)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.CORS)
	r.Use(rl.Middleware)

	// Health endpoints
	r.Get("/healthz", healthH.Healthz)
	r.Get("/readyz", healthH.Readyz)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authH.Register)
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.Refresh)
			r.Get("/google", authH.GoogleRedirect)
			r.Get("/google/callback", authH.GoogleCallback)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWT.Secret))

			// User
			r.Get("/auth/me", authH.Me)
			r.Put("/auth/me", authH.UpdateProfile)

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
