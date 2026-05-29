package domain

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type webhookUseCase struct {
	repo   IdempotencyRepository
	logger *zap.Logger
	queue  chan *WebhookPayload
}

// NewWebhookUseCase initializes the worker pool and returns the use-case implementation.
func NewWebhookUseCase(repo IdempotencyRepository, logger *zap.Logger, workerCount int, queueSize int) WebhookUseCase {
	uc := &webhookUseCase{
		repo:   repo,
		logger: logger,
		queue:  make(chan *WebhookPayload, queueSize),
	}

	// Spin up the background workers
	for i := 1; i <= workerCount; i++ {
		go uc.worker(i)
	}

	return uc
}

func (uc *webhookUseCase) ProcessWebhook(ctx context.Context, payload *WebhookPayload) error {
	// Attempt to acquire an atomic lock in Redis using the unique EventID
	isUnique, err := uc.repo.SetLock(ctx, payload.EventID)
	if err != nil {
		uc.logger.Error("failed to check idempotency lock", zap.Error(err), zap.String("event_id", payload.EventID))
		return err
	}

	if !isUnique {
		// Event is already being processed or has been processed recently
		return ErrDuplicateRequest
	}

	// Hand off to the buffered channel queue (non-blocking drop if queue is completely packed)
	select {
	case uc.queue <- payload:
		uc.logger.Info("webhook accepted and queued", zap.String("event_id", payload.EventID))
		return nil
	default:
		// Queue is full, release the lock so the client can safely retry later
		_ = uc.repo.ReleaseLock(ctx, payload.EventID)
		return ErrQueueFull
	}
}

// Background worker consuming payloads sequentially from the shared channel
func (uc *webhookUseCase) worker(workerID int) {
	uc.logger.Info("background worker started", zap.Int("worker_id", workerID))

	for payload := range uc.queue {
		uc.logger.Info("worker processing job", zap.Int("worker_id", workerID), zap.String("event_id", payload.EventID))

		// Simulate heavy processing task (e.g., calling government KYC verify systems or updating DB)
		time.Sleep(2 * time.Second)

		uc.logger.Info("worker successfully completed job", zap.Int("worker_id", workerID), zap.String("event_id", payload.EventID))
	}
}
