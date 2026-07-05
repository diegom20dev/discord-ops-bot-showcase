package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const optString = 3 // Discord option type: STRING

type choice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type option struct {
	Type        int      `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Required    bool     `json:"required,omitempty"`
	Choices     []choice `json:"choices,omitempty"`
}
type command struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Options     []option `json:"options,omitempty"`
}

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	appID := os.Getenv("DISCORD_APP_ID")
	guildID := os.Getenv("DISCORD_GUILD_ID") // optional: instant registration for testing

	if token == "" || appID == "" {
		log.Fatal("Set DISCORD_BOT_TOKEN and DISCORD_APP_ID (and optionally DISCORD_GUILD_ID)")
	}

	commands := []command{
		{Name: "capture", Description: "Save a quick note", Options: []option{
			{Type: optString, Name: "content", Description: "Note content", Required: true},
		}},
		{Name: "inbox", Description: "List your notes"},
		{Name: "delete", Description: "Delete a note", Options: []option{
			{Type: optString, Name: "id", Description: "Note ID", Required: true},
		}},
		{Name: "summarize", Description: "Summarize notes with AI", Options: []option{
			{Type: optString, Name: "ids", Description: "Comma-separated note IDs", Required: true},
		}},
		{Name: "remind", Description: "Set a reminder", Options: []option{
			{Type: optString, Name: "content", Description: "What to remember", Required: true},
			{Type: optString, Name: "in", Description: "When: e.g. 30m, 2h, 1d (default 1h)"},
		}},
		{Name: "export", Description: "Export your notes to a file", Options: []option{
			{Type: optString, Name: "format", Description: "Output format", Choices: []choice{
				{Name: "csv", Value: "csv"},
				{Name: "markdown", Value: "markdown"},
				{Name: "pdf", Value: "pdf"},
			}},
		}},
	}

	url := fmt.Sprintf("https://discord.com/api/v10/applications/%s/commands", appID)
	scope := "global"
	if guildID != "" {
		url = fmt.Sprintf("https://discord.com/api/v10/applications/%s/guilds/%s/commands", appID, guildID)
		scope = "guild"
	}

	body, _ := json.Marshal(commands) // PUT = bulk overwrite (idempotent)
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Fatalf("Discord API error %d: %s", resp.StatusCode, string(respBody))
	}
	fmt.Printf("Registered %d commands (%s)\n", len(commands), scope)
}
