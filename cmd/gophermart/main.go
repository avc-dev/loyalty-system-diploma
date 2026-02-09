package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/avc/loyalty-system-diploma/internal/app"
)

func main() {
	application, err := app.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
