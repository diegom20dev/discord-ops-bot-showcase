package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	httpClient *http.Client
	token      string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{},
		token:      token,
	}
}

func (c *Client) ReplyToInteraction(ctx context.Context, interactionID, interactionToken, message string) error {
	payload := map[string]interface{}{
		"type": 4,
		"data": map[string]interface{}{
			"content": message,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/interactions/%s/%s/callback", interactionID, interactionToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord api error: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) SendFollowUp(ctx context.Context, userID, message string) error {
	// Create DM channel
	dmPayload := map[string]string{
		"recipient_id": userID,
	}
	dmBody, err := json.Marshal(dmPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal DM payload: %w", err)
	}

	dmReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/v10/users/@me/channels", bytes.NewReader(dmBody))
	if err != nil {
		return fmt.Errorf("failed to create DM channel request: %w", err)
	}

	dmReq.Header.Set("Content-Type", "application/json")
	dmReq.Header.Set("Authorization", fmt.Sprintf("Bot %s", c.token))

	dmResp, err := c.httpClient.Do(dmReq)
	if err != nil {
		return fmt.Errorf("failed to create DM channel: %w", err)
	}
	defer dmResp.Body.Close()

	if dmResp.StatusCode >= 400 {
		body, _ := io.ReadAll(dmResp.Body)
		return fmt.Errorf("failed to create DM channel: %d %s", dmResp.StatusCode, string(body))
	}

	var dmReplyData struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(dmResp.Body).Decode(&dmReplyData); err != nil {
		return fmt.Errorf("failed to parse DM channel response: %w", err)
	}

	// Send message to DM channel
	msgPayload := map[string]string{
		"content": message,
	}
	msgBody, err := json.Marshal(msgPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}

	msgReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", dmReplyData.ID), bytes.NewReader(msgBody))
	if err != nil {
		return fmt.Errorf("failed to create message request: %w", err)
	}

	msgReq.Header.Set("Content-Type", "application/json")
	msgReq.Header.Set("Authorization", fmt.Sprintf("Bot %s", c.token))

	msgResp, err := c.httpClient.Do(msgReq)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer msgResp.Body.Close()

	if msgResp.StatusCode >= 400 {
		body, _ := io.ReadAll(msgResp.Body)
		return fmt.Errorf("failed to send message: %d %s", msgResp.StatusCode, string(body))
	}

	return nil
}
