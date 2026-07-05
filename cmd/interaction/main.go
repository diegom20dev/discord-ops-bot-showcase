package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/diegom20dev/discord-ops-bot/internal/app"
	"github.com/diegom20dev/discord-ops-bot/internal/discord"
	"github.com/diegom20dev/discord-ops-bot/internal/domain"
	"github.com/diegom20dev/discord-ops-bot/internal/platform"
)

var (
	validator            *discord.SignatureValidator
	createCaptureUC      *app.CreateCaptureUC
	listInboxUC          *app.ListInboxUC
	deleteCaptureUC      *app.DeleteCaptureUC
	summarizeMultipleUC  *app.SummarizeMultipleUC
	createReminderUC     *app.CreateReminderUC
	requestExportUC      *app.RequestExportUC
)

func init() {
	// Initialize Discord validator
	publicKey := os.Getenv("DISCORD_PUBLIC_KEY")
	if publicKey == "" {
		log.Fatal("DISCORD_PUBLIC_KEY not set")
	}
	validator = discord.NewSignatureValidator(publicKey)

	// Initialize AWS clients
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("Failed to load AWS config:", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)

	// Initialize adapters
	tableName := os.Getenv("CAPTURES_TABLE")
	queueURL := os.Getenv("TASKS_QUEUE_URL")

	repo := platform.NewDynamoDBRepository(ddbClient, tableName)
	queue := platform.NewSQSQueue(sqsClient, queueURL)
	rateLimiter := platform.NewDynamoDBRateLimiter(ddbClient, tableName)

	remindersTableName := os.Getenv("REMINDERS_TABLE")
	reminderRepo := platform.NewDynamoDBReminderRepository(ddbClient, remindersTableName)

	exportsTableName := os.Getenv("EXPORTS_TABLE")
	exportRepo := platform.NewDynamoDBExportRepository(ddbClient, exportsTableName)

	// Initialize use cases
	createCaptureUC = app.NewCreateCaptureUC(repo)
	listInboxUC = app.NewListInboxUC(repo)
	deleteCaptureUC = app.NewDeleteCaptureUC(repo)
	summarizeMultipleUC = app.NewSummarizeMultipleUC(repo, queue, rateLimiter)
	createReminderUC = app.NewCreateReminderUC(reminderRepo)
	requestExportUC = app.NewRequestExportUC(exportRepo, queue)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received request: %s %s", request.HTTPMethod, request.Path)

	// Parse interaction FIRST (before signature check, to handle PING)
	var interaction discord.InteractionRequest
	if err := json.Unmarshal([]byte(request.Body), &interaction); err != nil {
		log.Printf("Failed to parse interaction: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"Invalid payload"}`,
		}, nil
	}

	// Handle PING type (Discord validation) - respond immediately
	if interaction.Type == 1 {
		log.Printf("Responding to PING")
		response := map[string]int{"type": 1}
		body, _ := json.Marshal(response)
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       string(body),
		}, nil
	}

	// Verify Discord signature AFTER handling PING
	signature := request.Headers["x-signature-ed25519"]
	timestamp := request.Headers["x-signature-timestamp"]

	if err := validator.VerifySignature(signature, timestamp, request.Body); err != nil {
		log.Printf("Signature verification failed: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       `{"error":"Invalid signature"}`,
		}, nil
	}

	log.Printf("Signature verified, processing command")

	// Route command
	command, args := interaction.ParseCommand()
	switch command {
	case "capture":
		return handleCapture(ctx, interaction, args)
	case "inbox":
		return handleInbox(ctx, interaction)
	case "delete":
		return handleDelete(ctx, interaction, args)
	case "summarize":
		return handleSummarize(ctx, interaction, args)
	case "remind":
		return handleRemind(ctx, interaction, args)
	case "export":
		return handleExport(ctx, interaction, args)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf(`{"error":"Unknown command: %s"}`, command),
		}, nil
	}
}

func handleCapture(ctx context.Context, interaction discord.InteractionRequest, args map[string]string) (events.APIGatewayProxyResponse, error) {
	content, ok := args["content"]
	if !ok {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"Missing content argument"}`,
		}, nil
	}

	capture, err := createCaptureUC.Execute(ctx, app.CreateCaptureInput{
		UserID:  interaction.UserID(),
		Content: content,
	})
	if err != nil {
		log.Printf("Failed to create capture: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Failed to create capture"}`,
		}, nil
	}

	response := map[string]interface{}{
		"type": 4,
		"data": map[string]string{
			"content": fmt.Sprintf("✅ Note saved: **%s**\n\nUse `/summarize ids: %s` when you're ready to summarize it", capture.ID, capture.ID),
		},
	}
	body, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
	}, nil
}

func handleInbox(ctx context.Context, interaction discord.InteractionRequest) (events.APIGatewayProxyResponse, error) {
	captures, err := listInboxUC.Execute(ctx, interaction.UserID())
	if err != nil {
		log.Printf("Failed to list inbox: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Failed to list captures"}`,
		}, nil
	}

	// Filter out archived captures
	var active []*domain.Capture
	for _, c := range captures {
		if c.Status != "archived" {
			active = append(active, c)
		}
	}

	if len(active) == 0 {
		response := map[string]interface{}{
			"type": 4,
			"data": map[string]string{
				"content": "📋 Your inbox is empty",
			},
		}
		body, _ := json.Marshal(response)
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       string(body),
		}, nil
	}

	// Build embeds for each capture
	var embeds []map[string]interface{}
	for _, c := range active {
		color := 3447003 // Blue for pending
		if c.Status == "summarized" {
			color = 5763719 // Green
		}

		preview := c.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		embed := map[string]interface{}{
			"title":       c.ID,
			"description": preview,
			"color":       color,
			"fields": []map[string]interface{}{
				{
					"name":   "Status",
					"value":  c.Status,
					"inline": true,
				},
				{
					"name":   "Created",
					"value":  c.CreatedAt.Format("2006-01-02 15:04"),
					"inline": true,
				},
			},
		}

		if c.Summary != "" {
			embed["fields"] = append(embed["fields"].([]map[string]interface{}), map[string]interface{}{
				"name":   "Summary",
				"value":  c.Summary,
				"inline": false,
			})
		}

		embeds = append(embeds, embed)
	}

	response := map[string]interface{}{
		"type": 4,
		"data": map[string]interface{}{
			"content": fmt.Sprintf("📋 **Your Inbox** (%d items)", len(active)),
			"embeds":  embeds,
		},
	}
	body, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
	}, nil
}

func handleDelete(ctx context.Context, interaction discord.InteractionRequest, args map[string]string) (events.APIGatewayProxyResponse, error) {
	captureID, ok := args["id"]
	if !ok {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"Missing id argument"}`,
		}, nil
	}

	if err := deleteCaptureUC.Execute(ctx, interaction.UserID(), captureID); err != nil {
		log.Printf("Failed to delete capture: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error":"%v"}`, err),
		}, nil
	}

	response := map[string]interface{}{
		"type": 4,
		"data": map[string]string{
			"content": fmt.Sprintf("✅ Note **%s** deleted", captureID),
		},
	}
	body, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
	}, nil
}

func handleSummarize(ctx context.Context, interaction discord.InteractionRequest, args map[string]string) (events.APIGatewayProxyResponse, error) {
	idsArg, ok := args["ids"]
	if !ok {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"Missing ids argument"}`,
		}, nil
	}

	// Parse comma-separated IDs
	captureIDs := parseIDs(idsArg)
	if len(captureIDs) == 0 {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"No valid IDs provided"}`,
		}, nil
	}

	if err := summarizeMultipleUC.Execute(ctx, interaction.UserID(), captureIDs); err != nil {
		log.Printf("Failed to summarize captures: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error":"%v"}`, err),
		}, nil
	}

	response := map[string]interface{}{
		"type": 4,
		"data": map[string]string{
			"content": fmt.Sprintf("⏳ Summarizing %d note(s)... You'll receive the summaries shortly!", len(captureIDs)),
		},
	}
	body, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
	}, nil
}

func parseIDs(idsArg string) []string {
	var ids []string
	// Split by comma and trim spaces
	parts := bytes.Split([]byte(idsArg), []byte(","))
	for _, part := range parts {
		id := string(bytes.TrimSpace(part))
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func handleRemind(ctx context.Context, interaction discord.InteractionRequest, args map[string]string) (events.APIGatewayProxyResponse, error) {
	content, ok := args["content"]
	if !ok {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"Missing content argument"}`,
		}, nil
	}

	scheduleIn, ok := args["in"]
	if !ok {
		scheduleIn = "1h" // Default 1 hour
	}

	reminder, err := createReminderUC.Execute(ctx, app.CreateReminderInput{
		UserID:     interaction.UserID(),
		Content:    content,
		ScheduleIn: scheduleIn,
	})
	if err != nil {
		log.Printf("Failed to create reminder: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Failed to create reminder"}`,
		}, nil
	}

	response := map[string]interface{}{
		"type": 4,
		"data": map[string]string{
			"content": fmt.Sprintf("⏰ Reminder set for **%s** - \"%s\"", reminder.ScheduledAt.Format("2006-01-02 15:04"), reminder.Content),
		},
	}
	body, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
	}, nil
}

func handleExport(ctx context.Context, interaction discord.InteractionRequest, args map[string]string) (events.APIGatewayProxyResponse, error) {
	format, ok := args["format"]
	if !ok {
		format = "csv" // Default format
	}

	if format != "csv" && format != "markdown" && format != "pdf" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error":"Invalid format. Use: csv, markdown, or pdf"}`,
		}, nil
	}

	userID := interaction.UserID()
	log.Printf("Export request from user: %s (format: %s)", userID, format)

	_, err := requestExportUC.Execute(ctx, app.RequestExportInput{
		UserID: userID,
		Format: format,
	})
	if err != nil {
		log.Printf("Failed to request export: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Failed to request export"}`,
		}, nil
	}

	response := map[string]interface{}{
		"type": 4,
		"data": map[string]string{
			"content": fmt.Sprintf("⏳ Generating **%s** export... You'll receive it in a DM shortly!", format),
		},
	}
	body, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(body),
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatList(items []string) string {
	result := ""
	for i, item := range items {
		result += fmt.Sprintf("%d. %s\n", i+1, item)
	}
	return result
}

func main() {
	lambda.Start(handleRequest)
}
