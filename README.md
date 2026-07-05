# Discord Ops Bot

A serverless Discord bot built with hexagonal architecture, designed for capturing and summarizing operational notes.

## Architecture

```
discord-ops-bot/
├── cmd/                        # Lambda entry points (each is package main)
│   ├── interaction/main.go     # API Gateway → verifies signature, routes, enqueues
│   ├── worker/main.go          # SQS → async work → Discord follow-up
│   └── scheduler/main.go       # EventBridge cron → daily triage ping
├── internal/                   # Private, non-importable code
│   ├── domain/                 # Entities + ports (interfaces)
│   │   ├── models.go          # Capture, TriageEvent
│   │   └── ports.go           # Repository, Queue, DiscordClient, Summarizer
│   ├── app/                    # Use cases
│   │   ├── create_capture.go  # SaveCapture + enqueue summarization
│   │   ├── list_inbox.go      # List user's captures
│   │   └── summarize_capture.go
│   ├── discord/                # Discord protocol handling
│   │   ├── client.go          # API client for follow-ups
│   │   ├── signature.go       # Ed25519 signature verification
│   │   └── interaction.go     # Request/response types
│   └── platform/               # Adapter implementations
│       ├── dynamodb_repository.go  # DynamoDB persistence
│       └── sqs_queue.go           # SQS task queue
├── template.yaml               # SAM infrastructure-as-code
├── Makefile                    # Build & deploy
├── go.mod / go.sum
└── README.md
```

## Key Flows

### 1. Capture Creation (Discord → API Gateway → DynamoDB + SQS)
- User runs `/capture content:...` in Discord
- Signature verified with Ed25519
- Capture saved to DynamoDB
- Summarization task enqueued in SQS
- Immediate ACK response sent to Discord

### 2. Background Summarization (SQS → Worker Lambda → DynamoDB)
- Worker polls SQS queue (batches of 10)
- Retrieves capture from DynamoDB
- Generates summary (via Summarizer port)
- Updates capture status to "summarized"
- Sends follow-up message to user's Discord DM

### 3. Daily Triage (EventBridge → Scheduler → Discord)
- Runs daily at 9 AM UTC
- Queries all pending captures
- Generates digest message
- Posts to designated Discord channel

## Setup

### Prerequisites
- Go 1.25+
- AWS CLI configured
- SAM CLI
- Discord bot token & public key

### Deployment

```bash
# 1. Set environment
export DISCORD_PUBLIC_KEY="<your-public-key>"
export DISCORD_BOT_TOKEN="<your-bot-token>"

# 2. Build ARM64 binaries
make build

# 3. Deploy
make deploy
```

This creates:
- `discord-ops-captures` DynamoDB table
- `discord-ops-tasks` SQS queue
- Three Lambda functions (interaction, worker, scheduler)
- API Gateway endpoint (use as Discord bot endpoint)

## Development

### Run tests
```bash
make test
```

### Local testing
```bash
make local-test
```

Hits local endpoint at `http://localhost:3000/interactions`

## Adding Features

### New Use Case
1. Create `internal/app/your_feature.go`
2. Inject domain ports (interfaces) into use case
3. Register in Lambda handler

### New Port (External Dependency)
1. Define interface in `internal/domain/ports.go`
2. Implement adapter in `internal/platform/`
3. Inject into use cases

### New Command
1. Add case to `cmd/interaction/main.go` switch
2. Create corresponding use case
3. Update template.yaml if new permissions needed

## Dependencies

- `github.com/aws/aws-lambda-go` — Lambda runtime
- `github.com/aws/aws-sdk-go-v2/*` — AWS services
- Standard library for ED25519 signature validation

No external frameworks—clean hexagonal architecture with minimal dependencies.
