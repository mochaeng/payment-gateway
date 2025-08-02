package constants

type PaymentMode string

const (
	DefaultProcessorKey  PaymentMode = "default"
	FallbackProcessorKey PaymentMode = "fallback"
)
