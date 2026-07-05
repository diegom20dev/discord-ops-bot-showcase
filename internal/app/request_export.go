package app

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type RequestExportUC struct {
	exportRepo domain.ExportRepository
	queue      domain.Queue
}

func NewRequestExportUC(exportRepo domain.ExportRepository, queue domain.Queue) *RequestExportUC {
	return &RequestExportUC{
		exportRepo: exportRepo,
		queue:      queue,
	}
}

type RequestExportInput struct {
	UserID string
	Format string // csv, markdown, pdf
}

func (uc *RequestExportUC) Execute(ctx context.Context, input RequestExportInput) (*domain.Export, error) {
	if input.Format != "csv" && input.Format != "markdown" && input.Format != "pdf" {
		return nil, fmt.Errorf("invalid format: %s", input.Format)
	}

	export := &domain.Export{
		ID:        fmt.Sprintf("exp_%d_%d", time.Now().Unix(), rand.Intn(10000)),
		UserID:    input.UserID,
		Format:    input.Format,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := uc.exportRepo.SaveExport(ctx, export); err != nil {
		return nil, fmt.Errorf("failed to save export: %w", err)
	}

	// Enqueue export task
	task := &domain.Task{
		ID:        fmt.Sprintf("task_%d_%d", time.Now().Unix(), rand.Intn(10000)),
		Type:      "export",
		UserID:    input.UserID,
		CreatedAt: time.Now(),
		Retries:   0,
		Data: map[string]string{
			"export_id": export.ID,
			"format":    input.Format,
		},
	}

	if err := uc.queue.EnqueueTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to enqueue export task: %w", err)
	}

	return export, nil
}
