package platform

import (
	"context"
	"fmt"
	"sync"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

type BatchSummarizer struct {
	summarizer domain.Summarizer
	repo       domain.CaptureRepository
	batchSize  int
}

func NewBatchSummarizer(summarizer domain.Summarizer, repo domain.CaptureRepository, batchSize int) *BatchSummarizer {
	if batchSize <= 0 {
		batchSize = 10
	}
	return &BatchSummarizer{
		summarizer: summarizer,
		repo:       repo,
		batchSize:  batchSize,
	}
}

func (bs *BatchSummarizer) SummarizeBatch(ctx context.Context, captureIDs []string) error {
	if len(captureIDs) == 0 {
		return nil
	}

	// Fetch all captures
	captures := make(map[string]*domain.Capture)
	var contents []string
	var orderedIDs []string

	for _, id := range captureIDs {
		capture, err := bs.repo.GetCapture(ctx, id)
		if err != nil || capture == nil {
			continue
		}
		captures[id] = capture
		contents = append(contents, capture.Content)
		orderedIDs = append(orderedIDs, id)
	}

	if len(contents) == 0 {
		return fmt.Errorf("no valid captures found")
	}

	// Batch summarization - summarize multiple at once
	batchPrompt := bs.buildBatchPrompt(contents)
	summary, err := bs.summarizer.Summarize(ctx, batchPrompt)
	if err != nil {
		return fmt.Errorf("batch summarization failed: %w", err)
	}

	// Parse summaries and update captures
	summaries := bs.parseBatchSummaries(summary, len(contents))

	// Update captures in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(summaries))

	for i, id := range orderedIDs {
		if i >= len(summaries) {
			break
		}

		wg.Add(1)
		go func(captureID, summary string) {
			defer wg.Done()

			capture := captures[captureID]
			capture.Summary = summary
			capture.Status = "summarized"

			if err := bs.repo.UpdateCapture(ctx, capture); err != nil {
				errChan <- fmt.Errorf("failed to update capture %s: %w", captureID, err)
			}
		}(id, summaries[i])
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (bs *BatchSummarizer) buildBatchPrompt(contents []string) string {
	prompt := "Summarize each of the following operational notes concisely in 2-3 sentences each. Return each summary on a new line starting with '---'\n\n"

	for i, content := range contents {
		prompt += fmt.Sprintf("[Note %d]\n%s\n\n", i+1, content)
	}

	prompt += "Provide the summaries in the exact same order as the notes above, each starting with '---' on a new line."

	return prompt
}

func (bs *BatchSummarizer) parseBatchSummaries(response string, expectedCount int) []string {
	// Simple parser: split by '---' and extract summaries
	var summaries []string
	var current string

	for _, char := range response {
		if char == '-' {
			if current != "" {
				summaries = append(summaries, current)
				current = ""
			}
		} else if char != '\n' || current != "" {
			current += string(char)
		}
	}

	if current != "" {
		summaries = append(summaries, current)
	}

	// Ensure we have the right number
	if len(summaries) > expectedCount {
		summaries = summaries[:expectedCount]
	}

	for len(summaries) < expectedCount {
		summaries = append(summaries, "")
	}

	return summaries
}
