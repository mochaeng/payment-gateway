package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mochaeng/payment-gateway/internal/config"
	"github.com/mochaeng/payment-gateway/internal/constants"
	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/mochaeng/payment-gateway/internal/store"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

var (
	ErrProcessorsDown = errors.New("all processors are down")
	ErrQueueFull      = errors.New("queue is full")
)

type PaymentService struct {
	store      *store.RedisStore
	config     *config.Config
	health     *HealthMonitorService
	httpClient *fasthttp.Client
	queue      chan *models.QueuedPayment
}

func (p *PaymentService) Send(correlationID string, amount float64) error {
	return p.store.EnqueuePayment(&models.QueuedPayment{
		CorrelationID: correlationID,
		Amount:        amount,
	})
	// select {
	// case p.queue <- &models.QueuedPayment{
	// 	CorrelationID: correlationID,
	// 	Amount:        amount,
	// 	CreatedAt:     time.Now(),
	// }:
	// 	return nil
	// default:
	// 	return ErrQueueFull
	// }
}

func (p *PaymentService) processQueue() {
	for {
		payment, err := p.store.BlockingDequeuePayment(5 * time.Second)
		if err != nil {
			if err != redis.Nil {
				fmt.Printf("Failed to dequeue payment: %s\n", err)
			}
			continue
		}

		if err := p.tryProcess(payment); err != nil {
			payment.RetryCount++
			go func() {
				time.Sleep(time.Duration(payment.RetryCount) * time.Second)
				err := p.store.EnqueuePayment(payment)
				if err != nil {
					fmt.Printf("Failed to enqueue retried payment: %s", err)
				}
			}()
		}
	}

	// for payment := range p.queue {
	// 	if err := p.tryProcess(payment); err != nil {
	// 		fmt.Println(err)
	// 		payment.RetryCount++
	// 		if payment.RetryCount < 5 {
	// 			time.Sleep(time.Duration(payment.RetryCount) * time.Second)
	// 			p.queue <- payment
	// 		}
	// 	}
	// }
}

func (p *PaymentService) tryProcess(payment *models.QueuedPayment) error {
	defaultHealth, err := p.store.GetProcessorHealth(constants.DefaultProcessorKey)
	if err != nil {
		return fmt.Errorf("failed to get default processor health: %w", err)
	}

	fallbackHealth, err := p.store.GetProcessorHealth(constants.FallbackProcessorKey)
	if err != nil {
		return fmt.Errorf("failed to get fallback processor health: %w", err)
	}

	switch {
	case !defaultHealth.Failing &&
		(defaultHealth.MinResponseTime <= fallbackHealth.MinResponseTime+p.config.ProcessorThreshold):
		return p.processPayment(constants.DefaultProcessorKey, payment)
	case !fallbackHealth.Failing:
		return p.processPayment(constants.FallbackProcessorKey, payment)
	default:
		return ErrProcessorsDown
	}
}

func (p *PaymentService) processPayment(processor constants.PaymentMode, payment *models.QueuedPayment) error {
	url := p.config.Urls[processor].PaymentURL

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	paymentReq := models.PaymentProcessorRequest{
		CorrelationID: payment.CorrelationID,
		Amount:        payment.Amount,
		RequestedAt:   time.Now().UTC(),
	}

	reqBody, err := json.Marshal(paymentReq)
	if err != nil {
		return fmt.Errorf("failed to marshal payment request: %w", err)
	}

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(reqBody)

	if err := p.httpClient.Do(req, resp); err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}

	if resp.StatusCode() >= 500 {
		return fmt.Errorf("processor with status code [%d]", resp.StatusCode())
	}

	fmt.Printf("payment successed: %s with %f\n", processor, payment.Amount)
	p.store.UpdateSummary(processor, payment.Amount)

	return nil
}
