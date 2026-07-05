package app

import (
	"context"
	"fmt"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type DeleteCaptureUC struct {
	repo domain.CaptureRepository
}

func NewDeleteCaptureUC(repo domain.CaptureRepository) *DeleteCaptureUC {
	return &DeleteCaptureUC{repo: repo}
}

func (uc *DeleteCaptureUC) Execute(ctx context.Context, userID, captureID string) error {
	// Verify ownership
	capture, err := uc.repo.GetCapture(ctx, captureID)
	if err != nil {
		return fmt.Errorf("failed to get capture: %w", err)
	}

	if capture == nil {
		return fmt.Errorf("capture not found")
	}

	if capture.UserID != userID {
		return fmt.Errorf("unauthorized: capture belongs to another user")
	}

	// Delete by setting status to archived
	capture.Status = "archived"
	if err := uc.repo.UpdateCapture(ctx, capture); err != nil {
		return fmt.Errorf("failed to delete capture: %w", err)
	}

	return nil
}
