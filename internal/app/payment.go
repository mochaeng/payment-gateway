package app

import (
	"encoding/json"
	"fmt"

	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/valyala/fasthttp"
)

func (app *Application) paymentsHandler(ctx *fasthttp.RequestCtx) {
	fmt.Println("receiving payment request")

	var req models.PaymentRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(400)
		ctx.SetBodyString(`{"error":"Invalid JSON"}`)
		return
	}

	if req.CorrelationID == "" || req.Amount <= 0 {
		ctx.SetStatusCode(400)
		ctx.SetBodyString(`{"error":"Invalid request"}`)
		return
	}

	err := app.services.Payment.Send(req.CorrelationID, req.Amount)
	if err != nil {
		ctx.SetStatusCode(500)
		ctx.SetBodyString(`{"error":"Payment failed"}`)
		fmt.Println("failed to process:", err)
	} else {
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"message":"Payment processed"}`)
	}

}
