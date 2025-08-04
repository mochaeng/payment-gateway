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
			if payment.RetryCount < 3 {
				payment.RetryCount++

				backoffDuration := time.Duration(payment.RetryCount*payment.RetryCount) * time.Second

				go func(payment *models.QueuedPayment) {
					time.Sleep(backoffDuration)
					if err := p.store.EnqueuePayment(payment); err != nil {
						fmt.Printf("Failed to enqueue retried payment [%s] with [%s]\n",
							payment.CorrelationID, err)
					}
				}(payment)
			} else {
				fmt.Printf("Payment %s failed after %d retries, giving up\n",
					payment.CorrelationID, payment.RetryCount)
			}
		}
	}
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
	case !defaultHealth.Failing:
		return p.processPayment(constants.DefaultProcessorKey, payment)
	// case !defaultHealth.Failing &&
	// 	(defaultHealth.MinResponseTime <= fallbackHealth.MinResponseTime+p.config.ProcessorThreshold):
	// 	return p.processPayment(constants.DefaultProcessorKey, payment)
	case !fallbackHealth.Failing:
		return p.processPayment(constants.FallbackProcessorKey, payment)
	default:
		return ErrProcessorsDown
	}
}

func (p *PaymentService) processPayment(processor constants.PaymentMode, payment *models.QueuedPayment) error {
	processedKey := fmt.Sprintf("processed:%s", payment.CorrelationID)

	isSet, err := p.store.SetProcessedPayment(processedKey, processor, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to check payment processing status: %w", err)
	}

	if !isSet {
		fmt.Printf("payment [%s] already processed, skipping\n", payment.CorrelationID)
	}

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
		p.store.RemoveProcessedPayment(processedKey)
		return fmt.Errorf("failed to marshal payment request: %w", err)
	}

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(reqBody)

	if err := p.httpClient.Do(req, resp); err != nil {
		p.store.RemoveProcessedPayment(processedKey)
		return fmt.Errorf("failed to do request: %w", err)
	}

	if resp.StatusCode() >= 400 {
		p.store.RemoveProcessedPayment(processedKey)
		return fmt.Errorf("processor with status code [%d]", resp.StatusCode())
	}

	if err := p.store.UpdateSummary(processor, payment.Amount); err != nil {
		fmt.Printf("CRITICAL: failed to update summary for payment [%s] with value [%f]\n", payment.CorrelationID, payment.Amount)
		return fmt.Errorf("failed to update summary: %w", err)
	}

	fmt.Printf("payment successed: %s with %f\n", processor, payment.Amount)

	return nil
}
