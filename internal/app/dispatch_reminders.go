package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type DispatchRemindersUC struct {
	repo    domain.ReminderRepository
	discord domain.DiscordClient
}

func NewDispatchRemindersUC(repo domain.ReminderRepository, discord domain.DiscordClient) *DispatchRemindersUC {
	return &DispatchRemindersUC{repo: repo, discord: discord}
}

// Execute busca reminders pendientes cuyo scheduled_at ya pasó y los envía por DM.
// Retorna la cantidad de reminders despachados exitosamente.
func (uc *DispatchRemindersUC) Execute(ctx context.Context) (int, error) {
	reminders, err := uc.repo.ListPendingReminders(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list pending reminders: %w", err)
	}

	now := time.Now()
	dispatched := 0

	for _, reminder := range reminders {
		// Solo despachar los que ya vencieron
		if now.Before(reminder.ScheduledAt) {
			continue
		}

		message := fmt.Sprintf("🔔 **Reminder:** %s", reminder.Content)
		if err := uc.discord.SendFollowUp(ctx, reminder.UserID, message); err != nil {
			// No bloquear los demás si uno falla; se reintentará en la próxima corrida
			log.Printf("Failed to send reminder %s to user %s: %v", reminder.ID, reminder.UserID, err)
			continue
		}

		// Marcar como enviado para no reenviarlo
		reminder.Status = "sent"
		if err := uc.repo.UpdateReminder(ctx, reminder); err != nil {
			log.Printf("Failed to update reminder %s status: %v", reminder.ID, err)
			continue
		}

		dispatched++
	}

	return dispatched, nil
}
