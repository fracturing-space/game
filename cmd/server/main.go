package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fracturing-space/game/internal/cmd/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil)).With("service", "game")
	slog.SetDefault(logger)

	cfg, err := server.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		logger.Error("parse server config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, cfg); err != nil {
		logger.Error("serve game", "error", err)
		os.Exit(1)
	}
}
