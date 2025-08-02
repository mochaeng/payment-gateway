package app

import (
	"encoding/json"
	"time"

	"github.com/valyala/fasthttp"
)

func (app *Application) paymentsSummaryHandler(ctx *fasthttp.RequestCtx) {
	var from, to *time.Time

	fromStr := string(ctx.QueryArgs().Peek("from"))
	if fromStr != "" {
		fromTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			ctx.SetStatusCode(400)
			ctx.SetBodyString(`{"error":"Invalid 'from' timestamp format. Use ISO 8601 format"}`)
			return
		}
		from = &fromTime
	}

	toStr := string(ctx.QueryArgs().Peek("to"))
	if toStr != "" {
		toTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			ctx.SetStatusCode(400)
			ctx.SetBodyString(`{"error":"Invalid 'to' timestamp format. Use ISO 8601 format"}`)
			return
		}
		to = &toTime
	}

	if from != nil && to != nil && from.After(*to) {
		ctx.SetStatusCode(400)
		ctx.SetBodyString(`{"error":"'from' timestamp cannot be after 'to' timestamp"}`)
		return
	}

	summary, err := app.services.Summary.GetSummary(from, to)
	if err != nil {
		ctx.SetStatusCode(500)
		ctx.SetBodyString(`{"error":"Failed to get payment summary"}`)
		return
	}

	response, err := json.Marshal(summary)
	if err != nil {
		ctx.SetStatusCode(500)
		ctx.SetBodyString(`{"error":"Failed to serialize response"}`)
		return
	}

	ctx.SetStatusCode(200)
	ctx.SetBody(response)
}
