package models

import "time"

type PaymentRequest struct {
	CorrelationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

type PaymentProcessorRequest struct {
	CorrelationID string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestAt     time.Time `json:"requestedAt"`
}

type PaymentProcessorResponse struct {
	Message string `json:"message"`
}

type ProcessorSummary struct {
	TotalRequest int64   `json:"totalRequests"`
	TotalAmount  float64 `json:"totalAmount"`
}

type PaymentSummaryResponse struct {
	Default  ProcessorSummary `json:"default"`
	Fallback ProcessorSummary `json:"fallback"`
}
