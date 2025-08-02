package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mochaeng/payment-gateway/internal/constants"
	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	healthPrefix      = "health:"
	summaryPrefix     = "summary:"
	paymentPrefix     = "payments:"
	totalAmountPrefix = "total_amount:"
	totalCountPrefix  = "total_count:"

	paymentQueueKey = "payment_queue"
)

type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisStore(url string) (*RedisStore, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(opt)

	return &RedisStore{
		client: client,
		ctx:    context.Background(),
	}, nil
}

func (r *RedisStore) GetProcessorHealth(processor constants.PaymentMode) (*models.ProcessorHealth, error) {
	healthKey := fmt.Sprintf("%s%s", healthPrefix, processor)

	data, err := r.client.Get(r.ctx, healthKey).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get health processor: %w", err)
	}

	var health models.ProcessorHealth
	err = json.Unmarshal(data, &health)
	return &health, err
}

func (r *RedisStore) SetProcessorHealth(processor constants.PaymentMode, health models.ProcessorHealth) error {
	data, err := json.Marshal(health)
	if err != nil {
		return fmt.Errorf("failed to marshal health processor: %w", err)
	}

	healthKey := fmt.Sprintf("%s%s", healthPrefix, processor)

	return r.client.Set(r.ctx, healthKey, data, 0).Err()
}

func (r *RedisStore) EnqueuePayment(payment *models.QueuedPayment) error {
	data, err := json.Marshal(payment)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}

	return r.client.LPush(r.ctx, paymentQueueKey, data).Err()
}

func (r *RedisStore) DequeuePayment() (*models.QueuedPayment, error) {
	data, err := r.client.RPop(r.ctx, paymentQueueKey).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue payment: %w", err)
	}

	var payment models.QueuedPayment
	err = json.Unmarshal(data, &payment)
	return &payment, err
}

func (r *RedisStore) QueueSize() (int64, error) {
	return r.client.LLen(r.ctx, paymentQueueKey).Result()
}

func (r *RedisStore) UpdateSummary(processor constants.PaymentMode, amount float64) error {
	now := time.Now().UTC()
	timestamp := now.Unix()

	timeStampNano := now.UnixNano()

	recordsKey := fmt.Sprintf("%s%s:records", paymentPrefix, processor)

	pipe := r.client.Pipeline()

	paymentRecord := map[string]interface{}{
		"amount":    amount,
		"timestamp": timestamp,
	}

	recordData, err := json.Marshal(paymentRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal payment record: %w", err)
	}

	pipe.ZAdd(r.ctx, recordsKey, redis.Z{
		Score:  float64(timestamp),
		Member: fmt.Sprintf("%d:%s", timeStampNano, string(recordData)),
	})

	totalAmountKey := fmt.Sprintf("%s%s%s", summaryPrefix, totalAmountPrefix, processor)
	totalCountKey := fmt.Sprintf("%s%s%s", summaryPrefix, totalCountPrefix, processor)

	pipe.IncrByFloat(r.ctx, totalAmountKey, amount)
	pipe.Incr(r.ctx, totalCountKey)

	_, err = pipe.Exec(r.ctx)

	return err
}

func (r *RedisStore) GetSummary(from, to *time.Time) (*models.PaymentSummaryResponse, error) {
	response := &models.PaymentSummaryResponse{}

	defaultSummary, err := r.getProcecssorSummary(constants.DefaultProcessorKey, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get default processor summary: %w", err)
	}

	fallbackSummary, err := r.getProcecssorSummary(constants.FallbackProcessorKey, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get fallback processor summary: %w", err)
	}

	response.Default = *defaultSummary
	response.Fallback = *fallbackSummary

	return response, nil
}

func (r *RedisStore) getProcecssorSummary(processor constants.PaymentMode, from, to *time.Time) (*models.ProcessorSummary, error) {
	if from == nil && to == nil {
		return r.getTotalSummary(processor)
	}
	return r.getTimeFilteredSummary(processor, from, to)
}

func (r *RedisStore) getTotalSummary(processor constants.PaymentMode) (*models.ProcessorSummary, error) {
	totalAmountKey := fmt.Sprintf("%s%s%s", summaryPrefix, totalAmountPrefix, processor)
	totalCountKey := fmt.Sprintf("%s%s%s", summaryPrefix, totalCountPrefix, processor)

	pipe := r.client.Pipeline()
	amountCmd := pipe.Get(r.ctx, totalAmountKey)
	countCmd := pipe.Get(r.ctx, totalCountKey)

	_, err := pipe.Exec(r.ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	var totalAmount float64
	var totalCount int64

	if amountCmd.Err() == nil {
		totalAmount, _ = amountCmd.Float64()
	}

	if countCmd.Err() == nil {
		totalCount, _ = countCmd.Int64()
	}

	return &models.ProcessorSummary{
		TotalRequest: totalCount,
		TotalAmount:  totalAmount,
	}, nil
}

func (r *RedisStore) getTimeFilteredSummary(processor constants.PaymentMode, from, to *time.Time) (*models.ProcessorSummary, error) {
	recordsKey := fmt.Sprintf("%s%s:records", paymentPrefix, processor)

	var minScore, maxScore string

	if from != nil {
		minScore = fmt.Sprintf("%d", from.Unix())
	} else {
		minScore = "-inf"
	}

	if to != nil {
		maxScore = fmt.Sprintf("%d", to.Unix())
	} else {
		maxScore = "+inf"
	}

	records, err := r.client.ZRangeByScore(r.ctx, recordsKey, &redis.ZRangeBy{
		Min: minScore,
		Max: maxScore,
	}).Result()

	if err != nil && err != redis.Nil {
		return nil, err
	}

	var totalAmount float64
	totalCount := int64(len(records))

	for _, recordStr := range records {
		parts := strings.SplitN(recordStr, ":", 2)
		if len(parts) != 2 {
			continue
		}

		var record map[string]any
		if err := json.Unmarshal([]byte(parts[1]), &record); err != nil {
			continue
		}

		if amount, ok := record["amount"].(float64); ok {
			totalAmount += amount
		}
	}

	return &models.ProcessorSummary{
		TotalRequest: totalCount,
		TotalAmount:  totalAmount,
	}, nil
}
