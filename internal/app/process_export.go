package app

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jung-kurt/gofpdf/v2"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type ProcessExportUC struct {
	captureRepo domain.CaptureRepository
	exportRepo  domain.ExportRepository
	storage     domain.FileStorage
	discord     domain.DiscordClient
}

func NewProcessExportUC(captureRepo domain.CaptureRepository, exportRepo domain.ExportRepository, storage domain.FileStorage, discord domain.DiscordClient) *ProcessExportUC {
	return &ProcessExportUC{
		captureRepo: captureRepo,
		exportRepo:  exportRepo,
		storage:     storage,
		discord:     discord,
	}
}

func (uc *ProcessExportUC) Execute(ctx context.Context, exportID string, userID string, format string) error {
	export, err := uc.exportRepo.GetExport(ctx, exportID)
	if err != nil || export == nil {
		return fmt.Errorf("export not found: %s", exportID)
	}

	// Get captures
	captures, err := uc.captureRepo.ListCapturesByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get captures: %w", err)
	}

	var fileData []byte
	var fileName string

	switch format {
	case "csv":
		fileData = []byte(uc.generateCSV(captures))
		fileName = fmt.Sprintf("captures_%s.csv", exportID)
	case "markdown":
		fileData = []byte(uc.generateMarkdown(captures))
		fileName = fmt.Sprintf("captures_%s.md", exportID)
	case "pdf":
		pdfData, err := uc.generatePDF(captures)
		if err != nil {
			return fmt.Errorf("failed to generate PDF: %w", err)
		}
		fileData = pdfData
		fileName = fmt.Sprintf("captures_%s.pdf", exportID)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	// Upload to S3
	fileURL, err := uc.storage.Upload(ctx, fileName, fileData)
	if err != nil {
		export.Status = "failed"
		uc.exportRepo.UpdateExport(ctx, export)
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Update export status
	export.Status = "completed"
	export.FileURL = fileURL
	if err := uc.exportRepo.UpdateExport(ctx, export); err != nil {
		return fmt.Errorf("failed to update export: %w", err)
	}

	// Send to Discord
	message := fmt.Sprintf("📥 Your **%s** export is ready!\n%s", format, fileURL)
	if err := uc.discord.SendFollowUp(ctx, userID, message); err != nil {
		return fmt.Errorf("failed to send discord message: %w", err)
	}

	return nil
}

func (uc *ProcessExportUC) generateCSV(captures []*domain.Capture) string {
	csv := "ID,Created At,Content,Summary,Status\n"
	for _, c := range captures {
		csv += fmt.Sprintf("%s,%s,%q,%q,%s\n", c.ID, c.CreatedAt.Format("2006-01-02 15:04:05"), c.Content, c.Summary, c.Status)
	}
	return csv
}

func (uc *ProcessExportUC) generateMarkdown(captures []*domain.Capture) string {
	md := "# Discord Ops - Captured Notes Export\n\n"
	for _, c := range captures {
		md += fmt.Sprintf("## %s\n\n", c.ID)
		md += fmt.Sprintf("**Date:** %s\n", c.CreatedAt.Format("2006-01-02 15:04:05"))
		md += fmt.Sprintf("**Status:** %s\n\n", c.Status)
		md += fmt.Sprintf("**Content:**\n%s\n\n", c.Content)
		if c.Summary != "" {
			md += fmt.Sprintf("**Summary:**\n%s\n\n", c.Summary)
		}
		md += "---\n\n"
	}
	return md
}

func (uc *ProcessExportUC) generatePDF(captures []*domain.Capture) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Discord Ops - Captured Notes Export")
	pdf.Ln(15)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 10, fmt.Sprintf("Export Date: %s", "generated"))
	pdf.Ln(15)

	for _, c := range captures {
		// Title
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 10, c.ID)
		pdf.Ln(8)

		// Metadata
		pdf.SetFont("Arial", "", 9)
		pdf.Cell(0, 5, fmt.Sprintf("Date: %s", c.CreatedAt.Format("2006-01-02 15:04:05")))
		pdf.Ln(5)
		pdf.Cell(0, 5, fmt.Sprintf("Status: %s", c.Status))
		pdf.Ln(8)

		// Content
		pdf.SetFont("Arial", "", 10)
		pdf.MultiCell(0, 5, fmt.Sprintf("Content:\n%s", c.Content), "", "L", false)
		pdf.Ln(5)

		// Summary if available
		if c.Summary != "" {
			pdf.SetFont("Arial", "I", 10)
			pdf.MultiCell(0, 5, fmt.Sprintf("Summary:\n%s", c.Summary), "", "L", false)
			pdf.Ln(5)
		}

		// Separator
		pdf.SetDrawColor(200, 200, 200)
		pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
		pdf.Ln(10)
	}

	// Generate PDF to buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}
