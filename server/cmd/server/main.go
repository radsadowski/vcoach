package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// @title           vcoach API
// @version         1.0
// @description     Personal AI athletic training coach — chat-first, self-hosted.
// @description     The current athlete is resolved server-side (personal-first MVP); requests need no authentication headers yet.
//
// @host            localhost:8080
// @BasePath        /api
// @schemes         http https
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
