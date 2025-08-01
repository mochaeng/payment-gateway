package main

import (
	"log"

	"github.com/mochaeng/payment-gateway/internal/app"
	"github.com/mochaeng/payment-gateway/internal/config"
	"github.com/mochaeng/payment-gateway/internal/services"
)

func main() {
	cfg := config.Load()
	services := services.NewServices()

	app := app.NewApp(cfg, services)

	server := app.Mount()
	if err := app.Run(server); err != nil {
		log.Fatalf("Error starting server: %w", err)
	}
}
