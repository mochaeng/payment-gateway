package app

import (
	"encoding/json"

	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/valyala/fasthttp"
)

func handleSummary(ctx *fasthttp.RequestCtx) {
	response := models.PaymentSummaryResponse{}

	responseBytes, _ := json.Marshal(response)
	ctx.SetStatusCode(200)
	ctx.SetBody(responseBytes)
}
