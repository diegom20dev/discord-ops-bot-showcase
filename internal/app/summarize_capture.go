package app

import (
	"context"
	"fmt"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type SummarizeCaptureUC struct {
	repo       domain.CaptureRepository
	summarizer domain.Summarizer
	discord    domain.DiscordClient
}

func NewSummarizeCaptureUC(
	repo domain.CaptureRepository,
	summarizer domain.Summarizer,
	discord domain.DiscordClient,
) *SummarizeCaptureUC {
	return &SummarizeCaptureUC{
		repo:       repo,
		summarizer: summarizer,
		discord:    discord,
	}
}

func (uc *SummarizeCaptureUC) Execute(ctx context.Context, captureID string) error {
	capture, err := uc.repo.GetCapture(ctx, captureID)
	if err != nil {
		return fmt.Errorf("failed to get capture: %w", err)
	}

	summary, err := uc.summarizer.Summarize(ctx, capture.Content)
	if err != nil {
		return fmt.Errorf("failed to summarize: %w", err)
	}

	capture.Summary = summary
	capture.Status = "summarized"
	if err := uc.repo.UpdateCapture(ctx, capture); err != nil {
		return fmt.Errorf("failed to update capture: %w", err)
	}

	// Notify user
	if err := uc.discord.SendFollowUp(ctx, capture.UserID, summary); err != nil {
		return fmt.Errorf("failed to send follow-up: %w", err)
	}

	return nil
}
