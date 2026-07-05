package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/diegom20dev/discord-ops-bot/internal/app"
	"github.com/diegom20dev/discord-ops-bot/internal/discord"
	"github.com/diegom20dev/discord-ops-bot/internal/domain"
	"github.com/diegom20dev/discord-ops-bot/internal/platform"
)

const (
	maxRetries = 3
)

var (
	summarizeCaptureUC *app.SummarizeCaptureUC
	queue              domain.Queue
	batchSummarizer    *platform.BatchSummarizer
	processExportUC    *app.ProcessExportUC
)

func init() {
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
	queue = platform.NewSQSQueue(sqsClient, queueURL)

	// Initialize summarizer (Claude if available, otherwise noop)
	var summarizer domain.Summarizer
	claudeAPIKey := os.Getenv("CLAUDE_API_KEY")
	if claudeAPIKey == "" {
		log.Println("⚠️  CLAUDE_API_KEY not configured. Using noop summarizer.")
		log.Println("📝 To enable AI summarization, set CLAUDE_API_KEY environment variable.")
		summarizer = platform.NewNoopSummarizer()
	} else {
		summarizer = platform.NewClaudeSummarizer(claudeAPIKey, "claude-sonnet-4-6")
		log.Println("✅ Claude API summarizer enabled")
	}

	// Initialize Discord client
	discordToken := os.Getenv("DISCORD_BOT_TOKEN")
	if discordToken == "" {
		log.Println("⚠️  DISCORD_BOT_TOKEN not configured. DM notifications will not be sent.")
	}
	discordClient := discord.NewClient(discordToken)

	// Initialize batch summarizer
	batchSummarizer = platform.NewBatchSummarizer(summarizer, repo, 10)

	// Initialize S3 storage for exports
	s3Client := s3.NewFromConfig(cfg)
	exportsTableName := os.Getenv("EXPORTS_TABLE")
	exportsBucketName := os.Getenv("EXPORTS_BUCKET")
	exportRepo := platform.NewDynamoDBExportRepository(ddbClient, exportsTableName)
	storage := platform.NewS3Storage(s3Client, exportsBucketName, "")

	// Initialize use cases
	summarizeCaptureUC = app.NewSummarizeCaptureUC(repo, summarizer, discordClient)
	processExportUC = app.NewProcessExportUC(repo, exportRepo, storage, discordClient)
}

func handleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	// Group tasks by type
	var summarizeTasks []domain.Task
	var exportTasks []domain.Task
	var otherTasks []domain.Task

	for _, message := range sqsEvent.Records {
		var task domain.Task
		if err := json.Unmarshal([]byte(message.Body), &task); err != nil {
			log.Printf("Failed to unmarshal task: %v", err)
			continue
		}

		if task.Type == "summarize" {
			summarizeTasks = append(summarizeTasks, task)
		} else if task.Type == "export" {
			exportTasks = append(exportTasks, task)
		} else {
			otherTasks = append(otherTasks, task)
		}
	}

	// Process summarize tasks in batch
	if len(summarizeTasks) > 0 {
		processSummarizeBatch(ctx, summarizeTasks)
	}

	// Process export tasks
	for _, task := range exportTasks {
		processExportTask(ctx, task)
	}

	// Process other tasks individually
	for _, task := range otherTasks {
		log.Printf("Unknown task type: %s", task.Type)
	}

	return nil
}

func processSummarizeBatch(ctx context.Context, tasks []domain.Task) {
	captureIDs := make([]string, len(tasks))
	taskMap := make(map[string]domain.Task)

	for i, task := range tasks {
		captureIDs[i] = task.Data["capture_id"]
		taskMap[captureIDs[i]] = task
	}

	log.Printf("Processing batch of %d summarize tasks", len(captureIDs))

	if err := batchSummarizer.SummarizeBatch(ctx, captureIDs); err != nil {
		log.Printf("Batch summarization failed: %v", err)

		// Retry individual tasks
		for _, task := range tasks {
			if task.Retries < maxRetries {
				log.Printf("Requeuing task (retry %d/%d)", task.Retries, maxRetries)
				task.Retries++
				task.CreatedAt = time.Now()
				if requeueErr := queue.EnqueueTask(ctx, &task); requeueErr != nil {
					log.Printf("Failed to requeue task: %v", requeueErr)
				}
			}
		}
	} else {
		log.Printf("Batch summarization completed successfully")
	}
}

func processExportTask(ctx context.Context, task domain.Task) {
	exportID := task.Data["export_id"]
	format := task.Data["format"]

	log.Printf("Processing export task: %s (format: %s, userID: %s)", exportID, format, task.UserID)

	if err := processExportUC.Execute(ctx, exportID, task.UserID, format); err != nil {
		if task.Retries < maxRetries {
			log.Printf("Export failed (retry %d/%d): %v. Requeuing...", task.Retries, maxRetries, err)
			task.Retries++
			task.CreatedAt = time.Now()
			if requeueErr := queue.EnqueueTask(ctx, &task); requeueErr != nil {
				log.Printf("Failed to requeue export task: %v", requeueErr)
			}
		} else {
			log.Printf("Export failed after %d retries: %v", maxRetries, err)
		}
	} else {
		log.Printf("Export completed successfully")
	}
}

func main() {
	lambda.Start(handleRequest)
}
