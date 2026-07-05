package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type ExportCapturesUC struct {
	repo domain.CaptureRepository
}

func NewExportCapturesUC(repo domain.CaptureRepository) *ExportCapturesUC {
	return &ExportCapturesUC{repo: repo}
}

func (uc *ExportCapturesUC) ExportToCSV(ctx context.Context, userID string) (string, error) {
	captures, err := uc.repo.ListCapturesByUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to list captures: %w", err)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Header
	writer.Write([]string{"ID", "Created At", "Content", "Summary", "Status"})

	// Data
	for _, capture := range captures {
		writer.Write([]string{
			capture.ID,
			capture.CreatedAt.Format("2006-01-02 15:04:05"),
			capture.Content,
			capture.Summary,
			capture.Status,
		})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("failed to write CSV: %w", err)
	}

	return buf.String(), nil
}

func (uc *ExportCapturesUC) ExportToMarkdown(ctx context.Context, userID string) (string, error) {
	captures, err := uc.repo.ListCapturesByUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to list captures: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("# Discord Ops - Captured Notes Export\n\n")
	buf.WriteString(fmt.Sprintf("**User ID:** %s\n", userID))
	buf.WriteString(fmt.Sprintf("**Export Date:** %s\n\n", "generated"))

	for _, capture := range captures {
		buf.WriteString(fmt.Sprintf("## %s\n\n", capture.ID))
		buf.WriteString(fmt.Sprintf("**Date:** %s\n", capture.CreatedAt.Format("2006-01-02 15:04:05")))
		buf.WriteString(fmt.Sprintf("**Status:** %s\n\n", capture.Status))
		buf.WriteString(fmt.Sprintf("**Content:**\n%s\n\n", capture.Content))

		if capture.Summary != "" {
			buf.WriteString(fmt.Sprintf("**Summary:**\n%s\n\n", capture.Summary))
		}

		buf.WriteString("---\n\n")
	}

	return buf.String(), nil
}

// For PDF generation (basic)
func (uc *ExportCapturesUC) GeneratePDFReport(ctx context.Context, userID string) ([]byte, error) {
	markdown, err := uc.ExportToMarkdown(ctx, userID)
	if err != nil {
		return nil, err
	}

	// In production, use a PDF library like gofpdf
	// For now, return markdown as-is or a simple text representation
	return []byte(markdown), nil
}
