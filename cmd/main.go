package main

import (
	"log"

	"github.com/mochaeng/payment-gateway/internal/app"
	"github.com/mochaeng/payment-gateway/internal/config"
)

func main() {
	config := config.Load()

	app, err := app.NewApp(config)
	if err != nil {
		log.Fatalf("failed to create application: %s", err)
	}

	server := app.Mount()
	if err := app.Run(server); err != nil {
		log.Fatalf("Error starting server: %s", err)
	}
}
