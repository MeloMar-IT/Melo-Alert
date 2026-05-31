package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"signalhub/internal/config"
	"signalhub/internal/logger"
	"signalhub/internal/repository/postgres"
	"signalhub/internal/server"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	l := logger.New(cfg.Logging.Level)
	l.Info("Initializing signalhub")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	repo, err := postgres.NewRepository(ctx, cfg.Database.DSN)
	if err != nil {
		l.Error("failed to initialize repository", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	srv := server.New(cfg, l, repo)

	go func() {
		if err := srv.Start(); err != nil {
			l.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	l.Info("Graceful shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		l.Error("server shutdown failed", "error", err)
	}

	l.Info("Signalhub stopped")
}
