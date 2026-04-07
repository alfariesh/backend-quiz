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

	"github.com/rekanesiads/backend-quiz/config"
	"github.com/rekanesiads/backend-quiz/internal/handler"
	"github.com/rekanesiads/backend-quiz/internal/middleware"
	"github.com/rekanesiads/backend-quiz/internal/repository"
	"github.com/rekanesiads/backend-quiz/internal/service"
	appOtel "github.com/rekanesiads/backend-quiz/pkg/otel"
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

	// OpenTelemetry
	otelShutdown, err := appOtel.Setup(context.Background(), appOtel.Config{
		Enabled:     cfg.OTEL.Enabled,
		ExporterURL: cfg.OTEL.ExporterURL,
		ServiceName: cfg.OTEL.ServiceName,
	})
	if err != nil {
		return fmt.Errorf("setting up OpenTelemetry: %w", err)
	}
	defer otelShutdown(context.Background())

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

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.AccessDuration, cfg.JWT.RefreshDuration)
	oauthSvc := service.NewOAuthService(cfg.GoogleOAuth)
	mediaSvc := service.NewMediaService(cfg.R2)
	deckSvc := service.NewDeckService(deckRepo, cardRepo)
	cardSvc := service.NewCardService(cardRepo, deckRepo)
	studySvc := service.NewStudyService(cardRepo, reviewRepo, sessionRepo, userRepo, cfg.FSRS)
	statsSvc := service.NewStatsService(reviewRepo, sessionRepo, statsRepo, cardRepo)

	// Repositories (media)
	mediaRepo := repository.NewMediaRepository(pool)

	// Handlers
	healthH := handler.NewHealthHandler(pool)
	authH := handler.NewAuthHandler(authSvc, oauthSvc)
	mediaH := handler.NewMediaHandler(mediaSvc, mediaRepo)
	deckH := handler.NewDeckHandler(deckSvc)
	cardH := handler.NewCardHandler(cardSvc)
	studyH := handler.NewStudyHandler(studySvc)
	statsH := handler.NewStatsHandler(statsSvc)

	// Router
	r := chi.NewRouter()

	// Global middleware
	rl := middleware.NewRateLimiter(10, 20)
	if cfg.OTEL.Enabled {
		r.Use(middleware.Tracing(cfg.OTEL.ServiceName))
	}
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
			})

			// Study
			r.Route("/study", func(r chi.Router) {
				r.Post("/sessions", studyH.StartSession)
				r.Get("/sessions/{sessionID}", studyH.GetSession)
				r.Post("/sessions/{sessionID}/review", studyH.SubmitReview)
				r.Put("/sessions/{sessionID}/end", studyH.EndSession)
				r.Get("/preview/{deckID}", studyH.Preview)
			})

			// Stats
			r.Route("/stats", func(r chi.Router) {
				r.Get("/overview", statsH.Overview)
				r.Get("/heatmap", statsH.Heatmap)
				r.Get("/forecast", statsH.Forecast)
				r.Get("/deck/{deckID}", statsH.DeckStats)
			})

			// Media
			r.Route("/media", func(r chi.Router) {
				r.Post("/upload", mediaH.Upload)
				r.Delete("/{mediaID}", mediaH.Delete)
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
