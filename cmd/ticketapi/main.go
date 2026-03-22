package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"opspilot-go/internal/app/config"
	"opspilot-go/internal/app/logging"
	tickethttp "opspilot-go/internal/tools/http/tickets"
)

const defaultTicketAPIListenAddr = ":8090"

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)

	addr := os.Getenv("OPSPILOT_TICKET_API_LISTEN_ADDR")
	if addr == "" {
		addr = defaultTicketAPIListenAddr
	}

	logger.Info("ticket api listening", slog.String("addr", addr))
	if err := http.ListenAndServe(addr, tickethttp.NewFakeHandler(cfg.TicketAPIToken)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("ticket api failed", slog.Any("error", err))
		os.Exit(1)
	}
}
