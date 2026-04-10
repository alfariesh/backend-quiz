package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"

	"github.com/alfariesh/backend-quiz/config"
	"github.com/alfariesh/backend-quiz/internal/worker"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	zl := zerolog.New(os.Stdout).With().Timestamp().Logger()
	if cfg.Server.Environment == "development" {
		zl = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	}
	logger := slog.New(slogzerolog.Option{Logger: &zl}.NewZerologHandler())
	slog.SetDefault(logger)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()
	handlers := worker.NewTaskHandlers(logger)
	handlers.RegisterHandlers(mux)

	logger.Info("starting worker")
	if err := srv.Run(mux); err != nil {
		return fmt.Errorf("running worker: %w", err)
	}

	return nil
}
