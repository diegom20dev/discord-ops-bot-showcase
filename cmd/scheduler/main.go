package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

// SchedulerEvent es la estructura que envía EventBridge
type SchedulerEvent struct {
	DetailType string `json:"detail-type"`
	Source     string `json:"source"`
	Detail     map[string]interface{} `json:"detail"`
}

func handleRequest(ctx context.Context, event SchedulerEvent) error {
	log.Printf("Triage scheduler triggered: %s", event.DetailType)

	// TODO: Implement triage logic
	// - Query all pending captures
	// - Generate triage summary
	// - Send to Discord channel

	return nil
}

func main() {
	lambda.Start(handleRequest)
}
