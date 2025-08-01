package app

import (
	"log"

	"github.com/mochaeng/payment-gateway/internal/config"
	"github.com/mochaeng/payment-gateway/internal/services"
	"github.com/valyala/fasthttp"
)

type Application struct {
	config         *config.Config
	defaultClient  *fasthttp.Client
	fallbackClient *fasthttp.Client
	services       *services.Service
}

func NewApp(config *config.Config, services *services.Service) *Application {
	return &Application{
		config:   config,
		services: services,
	}
}

func (app *Application) Mount() *fasthttp.Server {
	return &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.Response.Header.Set("Content-Type", "application/json")

			switch string(ctx.Path()) {
			case "/payments":
				if ctx.IsPost() {
					app.paymentsHandler(ctx)
				} else {
					ctx.SetStatusCode(405)
					ctx.SetBodyString(`{"error":"Method not allowed"}`)
				}
			case "payments-summary":
				if ctx.IsGet() {
					// handleSummary()
				} else {
					ctx.SetStatusCode(405)
					ctx.SetBodyString(`{"error":"Method not allowed"}`)
				}
			default:
				ctx.SetStatusCode(404)
				ctx.SetBodyString(`{"error":"Not found"}`)
			}
		},
	}
}

func (app *Application) Run(server *fasthttp.Server) error {
	log.Printf("Starting server on port %s", app.config.Port)
	return server.ListenAndServe(":" + app.config.Port)
}
