package services

import (
	"time"

	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/mochaeng/payment-gateway/internal/store"
)

type SummaryService struct {
	store *store.RedisStore
}

func (s *SummaryService) GetSummary(from, to *time.Time) (*models.PaymentSummaryResponse, error) {
	return s.store.GetSummary(from, to)
}
