package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mochaeng/payment-gateway/internal/config"
	"github.com/mochaeng/payment-gateway/internal/constants"
	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/mochaeng/payment-gateway/internal/store"
	"github.com/valyala/fasthttp"
)

type HealthMonitorService struct {
	store       *store.RedisStore
	config      *config.Config
	lastChecked time.Time
	httpClient  *fasthttp.Client
}

func (m *HealthMonitorService) Start() {
	go m.monitorLoop()
}

func (m *HealthMonitorService) monitorLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if time.Since(m.lastChecked) > m.config.HealthCheckInterval {
			fmt.Println("Cheking all health systems")

			m.checkProcessor(constants.DefaultProcessorKey)
			m.checkProcessor(constants.FallbackProcessorKey)

			m.lastChecked = time.Now()
		}
	}
}

func (m *HealthMonitorService) checkProcessor(processor constants.PaymentMode) error {
	url := m.config.Urls[processor].HealthURL

	// fmt.Println("Healthy url: ", url)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")

	// ctx, cancel := context.WithTimeout(context.Background(), monitor.config.RequestTimeout)
	// defer cancel()

	err := m.httpClient.Do(req, resp)
	if err != nil {
		m.store.SetProcessorHealth(processor, models.ProcessorHealth{
			Failing:     true,
			LastChecked: time.Now(),
		})
		return err
	}

	var healthResp models.HealthResponse
	if err := json.Unmarshal(resp.Body(), &healthResp); err != nil {
		m.store.SetProcessorHealth(processor, models.ProcessorHealth{
			Failing:     true,
			LastChecked: time.Now(),
		})
		return err
	}

	m.store.SetProcessorHealth(processor, models.ProcessorHealth{
		Failing:         healthResp.Failing,
		MinResponseTime: healthResp.MinResponseTime,
		LastChecked:     time.Now(),
	})

	return nil
}
