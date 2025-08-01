package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mochaeng/payment-gateway/internal/constants"
	"github.com/mochaeng/payment-gateway/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	healthPrefix  = "health:"
	summaryPrefix = "summary:"

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
	err = json.Unmarshal(data, &healthKey)
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

func (r *RedisStore) UpdateSummary(processor constants.PaymentMode, amount float64) {}

func (r *RedisStore) GetSummary() {}
