package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"opspilot-go/internal/app/config"
	"opspilot-go/internal/app/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("worker booted", slog.String("env", cfg.Env))
	<-ctx.Done()
	logger.Info("worker shutdown complete")
}
