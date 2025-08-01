package services

import (
	"github.com/mochaeng/payment-gateway/internal/config"
	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/mochaeng/payment-gateway/internal/store"
	"github.com/valyala/fasthttp"
)

type Service struct {
	Payment interface {
		Send(correlationID string, amount float64) error
	}
	Health interface {
		Start()
	}
}

func NewServices(config *config.Config, store *store.RedisStore) *Service {
	health := HealthMonitorService{
		config:     config,
		store:      store,
		httpClient: &fasthttp.Client{},
	}

	payment := PaymentService{
		config:     config,
		store:      store,
		health:     &health,
		httpClient: &fasthttp.Client{},
		queue:      make(chan *models.QueuedPayment, config.MaxQueueSize),
	}

	health.Start()
	go payment.processQueue()

	return &Service{
		Payment: &payment,
		Health:  &health,
	}
}
