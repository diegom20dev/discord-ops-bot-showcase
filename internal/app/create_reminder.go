package app

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type CreateReminderUC struct {
	repo domain.ReminderRepository
}

func NewCreateReminderUC(repo domain.ReminderRepository) *CreateReminderUC {
	return &CreateReminderUC{repo: repo}
}

type CreateReminderInput struct {
	UserID    string
	Content   string
	ScheduleIn string // "1h", "30m", "2d", etc.
}

func (uc *CreateReminderUC) Execute(ctx context.Context, input CreateReminderInput) (*domain.Reminder, error) {
	duration, err := parseDuration(input.ScheduleIn)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule format: %w", err)
	}

	now := time.Now()
	scheduledAt := now.Add(duration)

	reminder := &domain.Reminder{
		ID:          fmt.Sprintf("rem_%d_%d", now.Unix(), rand.Intn(10000)),
		UserID:      input.UserID,
		Content:     input.Content,
		ScheduledAt: scheduledAt,
		CreatedAt:   now,
		Status:      "pending",
		TTL:         scheduledAt.Add(24 * time.Hour).Unix(), // Keep 24h after scheduled
	}

	if err := uc.repo.SaveReminder(ctx, reminder); err != nil {
		return nil, fmt.Errorf("failed to save reminder: %w", err)
	}

	return reminder, nil
}

func parseDuration(s string) (time.Duration, error) {
	// Parse "1h", "30m", "2d", etc.
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}

	// Try with 'd' for days
	if len(s) > 0 && s[len(s)-1] == 'd' {
		days := s[:len(s)-1]
		var numDays int
		if _, err := fmt.Sscanf(days, "%d", &numDays); err == nil {
			return time.Duration(numDays) * 24 * time.Hour, nil
		}
	}

	return 0, fmt.Errorf("invalid duration: %s", s)
}
