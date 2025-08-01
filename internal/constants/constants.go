package constants

type PaymentMode string

var (
	DefaultProcessorKey  PaymentMode = "default"
	FallbackProcessorKey PaymentMode = "fallback"
)
