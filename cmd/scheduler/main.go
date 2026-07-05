package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/diegom20dev/discord-ops-bot/internal/app"
	"github.com/diegom20dev/discord-ops-bot/internal/discord"
	"github.com/diegom20dev/discord-ops-bot/internal/platform"
)

var (
	dispatchRemindersUC *app.DispatchRemindersUC
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("Failed to load AWS config:", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)

	// Reminder repository
	remindersTableName := os.Getenv("REMINDERS_TABLE")
	reminderRepo := platform.NewDynamoDBReminderRepository(ddbClient, remindersTableName)

	// Discord client for DMs
	discordToken := os.Getenv("DISCORD_BOT_TOKEN")
	if discordToken == "" {
		log.Println("⚠️  DISCORD_BOT_TOKEN not configured. Reminder DMs will not be sent.")
	}
	discordClient := discord.NewClient(discordToken)

	dispatchRemindersUC = app.NewDispatchRemindersUC(reminderRepo, discordClient)
}

// SchedulerEvent es la estructura que envía EventBridge
type SchedulerEvent struct {
	DetailType string                 `json:"detail-type"`
	Source     string                 `json:"source"`
	Detail     map[string]interface{} `json:"detail"`
}

func handleRequest(ctx context.Context, event SchedulerEvent) error {
	log.Printf("Scheduler triggered: %s", event.DetailType)

	dispatched, err := dispatchRemindersUC.Execute(ctx)
	if err != nil {
		log.Printf("Failed to dispatch reminders: %v", err)
		return err
	}

	log.Printf("Dispatched %d reminder(s)", dispatched)
	return nil
}

func main() {
	lambda.Start(handleRequest)
}
