package config

import (
	"os"
	"time"
)

type Config struct {
	Port                 string
	RedisURL             string
	DefaultProcessorURL  string
	FallbackProcessorURL string
	HealthCheckInterval  time.Duration
	RequestTimeout       time.Duration
	MaxQueueSize         int
	ProcessorThreshold   int
}

func Load() *Config {
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		RedisURL:             getEnv("REDIS_URL", "localhost:6379"),
		DefaultProcessorURL:  getEnv("DEFAULT_PROCESSOR_URL", "http://localhost:8001"),
		FallbackProcessorURL: getEnv("FALLBACK_PROCESSOR_URL", "http://localhost:8002"),
		HealthCheckInterval:  parseDuration(getEnv("HEALTH_CHECK_INTERVAL", "5s")),
		RequestTimeout:       parseDuration(getEnv("REQUEST_TIMEOUT", "2s")),
		MaxQueueSize:         1000,
		ProcessorThreshold:   300,
	}
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
