package services

import (
	"time"

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
	Summary interface {
		GetSummary(from, to *time.Time) (*models.PaymentSummaryResponse, error)
	}
}

func NewServices(config *config.Config, store *store.RedisStore) *Service {
	health := HealthMonitorService{
		config:     config,
		store:      store,
		httpClient: &fasthttp.Client{},
	}
	health.Start()

	payment := PaymentService{
		config:     config,
		store:      store,
		health:     &health,
		httpClient: &fasthttp.Client{},
		queue:      make(chan *models.QueuedPayment, config.MaxQueueSize),
	}
	go payment.processQueue()

	summary := SummaryService{
		store: store,
	}

	return &Service{
		Payment: &payment,
		Health:  &health,
		Summary: &summary,
	}
}
