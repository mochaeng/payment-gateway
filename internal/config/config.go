package config

import (
	"os"
	"time"

	"github.com/mochaeng/payment-gateway/internal/constants"
)

type Config struct {
	Port                string
	RedisURL            string
	HealthCheckInterval time.Duration
	RequestTimeout      time.Duration
	MaxQueueSize        int
	ProcessorThreshold  int
	Urls                map[constants.PaymentMode]*ProcessorsConfig
}

type ProcessorsConfig struct {
	BaseURL    string
	PaymentURL string
	HealthURL  string
}

func Load() *Config {
	config := &Config{
		Port:                getEnv("PORT", "8080"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		HealthCheckInterval: parseDuration(getEnv("HEALTH_CHECK_INTERVAL", "5s")),
		RequestTimeout:      parseDuration(getEnv("REQUEST_TIMEOUT", "2s")),
		MaxQueueSize:        1000,
		ProcessorThreshold:  300,
		Urls:                make(map[constants.PaymentMode]*ProcessorsConfig, 2),
	}

	defaultBase := getEnv("DEFAULT_PROCESSOR_URL", "http://localhost:8001")
	fallbackbase := getEnv("FALLBACK_PROCESSOR_URL", "http://localhost:8002")
	config.Urls[constants.DefaultProcessorKey] = &ProcessorsConfig{
		BaseURL:    defaultBase,
		PaymentURL: defaultBase + "/payments",
		HealthURL:  defaultBase + "/payments/service-health",
	}
	config.Urls[constants.FallbackProcessorKey] = &ProcessorsConfig{
		BaseURL:    fallbackbase,
		PaymentURL: fallbackbase + "/payments",
		HealthURL:  fallbackbase + "/payments/service-health",
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	duration, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second
	}
	return duration
}
