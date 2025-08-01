package app

import (
	"encoding/json"

	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/valyala/fasthttp"
)

func handleSummary(ctx *fasthttp.RequestCtx) {
	response := models.PaymentSummaryResponse{
		Default: models.ProcessorSummary{
			TotalRequest: defaultCount,
			TotalAmount:  defaultAmount,
		},
		Fallback: models.ProcessorSummary{
			TotalRequest: fallbackCount,
			TotalAmount:  float64(fallbackCount),
		},
	}

	responseBytes, _ := json.Marshal(response)
	ctx.SetStatusCode(200)
	ctx.SetBody(responseBytes)
}
