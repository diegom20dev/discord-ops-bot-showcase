package platform

import (
	"context"
	"fmt"
)

type NoopSummarizer struct {
	message string
}

func NewNoopSummarizer() *NoopSummarizer {
	return &NoopSummarizer{
		message: "⚠️ Claude API not configured. To enable summarization, set CLAUDE_API_KEY environment variable with your Anthropic API key (get it from https://console.anthropic.com/account/keys)",
	}
}

func (ns *NoopSummarizer) Summarize(ctx context.Context, content string) (string, error) {
	return fmt.Sprintf("%s\n\n**Captured:** %s", ns.message, content[:min(100, len(content))]), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
