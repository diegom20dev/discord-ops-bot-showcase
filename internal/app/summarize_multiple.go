package app

import (
	"context"
	"fmt"
	"time"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type SummarizeMultipleUC struct {
	repo        domain.CaptureRepository
	queue       domain.Queue
	rateLimiter domain.RateLimiter
}

func NewSummarizeMultipleUC(repo domain.CaptureRepository, queue domain.Queue, rateLimiter domain.RateLimiter) *SummarizeMultipleUC {
	return &SummarizeMultipleUC{repo: repo, queue: queue, rateLimiter: rateLimiter}
}

func (uc *SummarizeMultipleUC) Execute(ctx context.Context, userID string, captureIDs []string) error {
	// Check rate limit
	allowed, err := uc.rateLimiter.AllowAction(ctx, userID, "summarize")
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		return fmt.Errorf("rate limit exceeded: max 10 summarizations per hour")
	}

	// Verify all captures belong to user and enqueue them
	for _, captureID := range captureIDs {
		capture, err := uc.repo.GetCapture(ctx, captureID)
		if err != nil {
			return fmt.Errorf("failed to get capture %s: %w", captureID, err)
		}

		if capture == nil {
			return fmt.Errorf("capture %s not found", captureID)
		}

		if capture.UserID != userID {
			return fmt.Errorf("unauthorized: capture %s belongs to another user", captureID)
		}

		// Enqueue summarization task
		task := &domain.Task{
			ID:        fmt.Sprintf("task_%d_%d", capture.CreatedAt.Unix(), len(captureID)),
			Type:      "summarize",
			UserID:    userID,
			CreatedAt: time.Now(),
			Retries:   0,
			Data: map[string]string{
				"capture_id": captureID,
			},
		}

		if err := uc.queue.EnqueueTask(ctx, task); err != nil {
			return fmt.Errorf("failed to enqueue task for %s: %w", captureID, err)
		}
	}

	return nil
}
