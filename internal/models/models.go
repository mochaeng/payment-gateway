package models

import "time"

type PaymentRequest struct {
	CorrelationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

type PaymentProcessorRequest struct {
	CorrelationID string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
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

type ProcessorHealth struct {
	Failing         bool
	MinResponseTime int
	LastChecked     time.Time
}

type QueuedPayment struct {
	CorrelationID string
	Amount        float64
	CreatedAt     time.Time
	RetryCount    int
}

type HealthResponse struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}
