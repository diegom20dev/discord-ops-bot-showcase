package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ClaudeSummarizer struct {
	apiKey         string
	model          string
	circuitBreaker *CircuitBreaker
}

func NewClaudeSummarizer(apiKey, model string) *ClaudeSummarizer {
	return &ClaudeSummarizer{
		apiKey:         apiKey,
		model:          model,
		circuitBreaker: NewCircuitBreaker(5, 60*time.Second), // 5 failures, 60s timeout
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MessageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type MessageResponse struct {
	Content []ContentBlock `json:"content"`
}

func (cs *ClaudeSummarizer) Summarize(ctx context.Context, content string) (string, error) {
	var result string
	err := cs.circuitBreaker.Call(func() error {
		var err error
		result, err = cs.callClaudeAPI(ctx, content)
		return err
	})
	return result, err
}

func (cs *ClaudeSummarizer) callClaudeAPI(ctx context.Context, content string) (string, error) {
	prompt := fmt.Sprintf(`Please summarize the following operational note concisely in 2-3 sentences:

"%s"

Focus on the key information and action items.`, content)

	req := MessageRequest{
		Model:     cs.model,
		MaxTokens: 500,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", cs.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call Claude API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var msgResp MessageResponse
	if err := json.Unmarshal(respBody, &msgResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w, body: %s", err, string(respBody))
	}

	if len(msgResp.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude API: %s", string(respBody))
	}

	return msgResp.Content[0].Text, nil
}
