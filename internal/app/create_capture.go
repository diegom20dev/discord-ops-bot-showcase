package app

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type CreateCaptureUC struct {
	repo domain.CaptureRepository
}

func NewCreateCaptureUC(repo domain.CaptureRepository) *CreateCaptureUC {
	return &CreateCaptureUC{repo: repo}
}

type CreateCaptureInput struct {
	UserID  string
	Content string
}

func (uc *CreateCaptureUC) Execute(ctx context.Context, input CreateCaptureInput) (*domain.Capture, error) {
	capture := &domain.Capture{
		ID:        fmt.Sprintf("cap_%d_%d", time.Now().Unix(), rand.Intn(10000)),
		UserID:    input.UserID,
		Content:   input.Content,
		CreatedAt: time.Now(),
		Status:    "pending",
	}

	if err := uc.repo.SaveCapture(ctx, capture); err != nil {
		return nil, fmt.Errorf("failed to save capture: %w", err)
	}

	return capture, nil
}
