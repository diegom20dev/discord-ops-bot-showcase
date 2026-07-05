package app

import (
	"context"
	"fmt"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type ListInboxUC struct {
	repo domain.CaptureRepository
}

func NewListInboxUC(repo domain.CaptureRepository) *ListInboxUC {
	return &ListInboxUC{repo: repo}
}

func (uc *ListInboxUC) Execute(ctx context.Context, userID string) ([]*domain.Capture, error) {
	captures, err := uc.repo.ListCapturesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list captures: %w", err)
	}
	return captures, nil
}
