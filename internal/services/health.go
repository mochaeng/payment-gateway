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

func (monitor *HealthMonitorService) Start() {
	go monitor.monitorLoop()
}

func (monitor *HealthMonitorService) monitorLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if time.Since(monitor.lastChecked) > monitor.config.HealthCheckInterval {
			monitor.checkProcessor(constants.DefaultProcessorKey)
			monitor.checkProcessor(constants.FallbackProcessorKey)
		}
	}
}

func (monitor *HealthMonitorService) checkProcessor(processor constants.PaymentMode) {
	url := fmt.Sprintf("%s%s", processor, "/payments/service-health")

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")

	// ctx, cancel := context.WithTimeout(context.Background(), monitor.config.RequestTimeout)
	// defer cancel()

	err := monitor.httpClient.Do(req, resp)
	if err != nil {
		monitor.store.SetProcessorHealth(processor, models.ProcessorHealth{
			Failing:     true,
			LastChecked: time.Now(),
		})
		return
	}

	var healthResp models.HealthResponse
	if err := json.Unmarshal(resp.Body(), &healthResp); err != nil {
		monitor.store.SetProcessorHealth(processor, models.ProcessorHealth{
			Failing:     true,
			LastChecked: time.Now(),
		})
		return
	}

	monitor.store.SetProcessorHealth(processor, models.ProcessorHealth{
		Failing:         healthResp.Failing,
		MinResponseTime: healthResp.MinResponseTime,
		LastChecked:     time.Now(),
	})
}
