# Discord Ops Bot

[![CI](https://github.com/diegom20dev/discord-ops-bot-showcase/actions/workflows/ci.yml/badge.svg)](https://github.com/diegom20dev/discord-ops-bot-showcase/actions/workflows/ci.yml)

A personal operations bot for Discord, built **100% serverless on AWS in Go** as a backend / system-design showcase. Capture notes, summarize them with **Claude (optional and pluggable)**, set reminders, and export your data — all through slash commands, on an **event-driven, hexagonal** architecture.

## Overview

The bot turns Discord slash commands into a small ops system. Fast commands answer instantly; slow work (AI summarization, file exports) is offloaded to a queue and a worker, then delivered back to the user by DM. AI is **optional**: if no Claude API key is configured, the bot degrades gracefully via a no-op summarizer instead of failing.

The domain is simple, but the engineering is production-grade: **three independent Lambdas**, a pure domain core with **ports & adapters**, a **circuit breaker** around the AI calls, per-user **rate limiting**, and **retry** handling on async tasks.

## Architecture

Serverless + event-driven + **hexagonal**. Three Lambdas share a framework-free domain core (interfaces = ports); infrastructure (DynamoDB, SQS, S3, Claude, Discord) is plugged in as adapters.

```
                          ┌──────────── AWS ────────────────────────────────┐
 Discord ──slash──► API Gateway ──► λ interaction ──► DynamoDB (captures…)   │
   ▲   ▲                              │  (verify Ed25519, route)             │
   │   │                             │▼ async cmds                          │
   │   │                             SQS ──► λ worker ──► Claude / Noop      │
   │   └──── DM (bot token) ◄─────────┘         │        └► S3 (exports)     │
   │                                            ▼                            │
   └──── DM ◄──── λ scheduler ◄──── EventBridge (cron) ── DynamoDB (reminders)
                          └───────────────────────────────────────────────────┘
```

| Layer | Folder | Responsibility |
|---|---|---|
| **Domain** | `internal/domain` | Pure models + **ports** (interfaces). No AWS, no Discord. |
| **Application** | `internal/app` | Use-cases (CreateCapture, SummarizeMultiple, RequestExport…). |
| **Infrastructure** | `internal/platform` | Adapters: DynamoDB, SQS, S3, Claude/Noop summarizer, circuit breaker, rate limiter. |
| **Entrypoints** | `cmd/{interaction,worker,scheduler}` | One Lambda binary each. |

## Slash Commands

| Command | Description | Flow |
|---|---|---|
| `/capture content:<text>` | Save a quick note | sync |
| `/inbox` | List your notes (status, date, summary) | sync |
| `/delete id:<id>` | Delete a note | sync |
| `/summarize ids:<id,id,…>` | Summarize one or more notes with AI | **async → DM** |
| `/remind content:<text> in:<1h>` | Set a reminder (default 1h) | sync · delivered by scheduler |
| `/export format:<csv\|markdown\|pdf>` | Export your notes to a file | **async → DM** |

## Engineering Patterns

### Three Lambdas, one event-driven system
- **interaction** — API Gateway HTTP handler: verifies the Discord Ed25519 signature, answers Discord's `PING`, routes commands, and enqueues async work.
- **worker** — SQS-triggered: groups tasks by type, runs AI summarization (batched) and exports, and DMs the result.
- **scheduler** — EventBridge cron: scans due reminders and DMs them.

### Optional / pluggable AI — the `Summarizer` port
AI is behind an interface, so the app doesn't depend on any provider — or on AI existing at all:

```go
type Summarizer interface {
    Summarize(ctx context.Context, content string) (string, error)
}
// worker picks the adapter at startup:
if os.Getenv("CLAUDE_API_KEY") == "" {
    summarizer = platform.NewNoopSummarizer()   // graceful degradation
} else {
    summarizer = platform.NewClaudeSummarizer(key, "claude-sonnet-4-6")
}
```

Without a key, `/summarize` returns a friendly "AI not configured" note instead of erroring. Swapping Claude for another model = one adapter, zero changes to the domain.

### Resilience — circuit breaker on the AI calls
`ClaudeSummarizer` wraps every request in a **circuit breaker** (closed → open → half-open; opens after 5 failures, resets after 60s) plus a 30s HTTP timeout. If the AI provider is down, the breaker fails fast instead of piling up hung requests.

### Async processing with retries
The worker consumes SQS, and on failure **requeues the task** with an incremented retry counter (max 3) before giving up — a terminal, observable outcome rather than a silent hang.

### Per-user rate limiting & hexagonal ports
Actions are rate-limited per user (DynamoDB-backed). The `Queue` port (`EnqueueTask`) is implemented by an SQS adapter, but could be swapped for RabbitMQ/Redis without touching use-cases — ports & adapters throughout.

## Tech Stack

| Technology | Role |
|---|---|
| **Go** | All Lambdas (arm64, `provided.al2023`, `bootstrap` binary) |
| **AWS Lambda** | interaction · worker · scheduler |
| **API Gateway** | Discord interactions endpoint |
| **DynamoDB** | captures, reminders (with TTL), exports, rate limits |
| **SQS** | async task queue |
| **S3** | export file storage |
| **EventBridge** | cron for reminders |
| **Anthropic Claude API** | optional AI summarization |
| **AWS SAM** | Infrastructure as Code (`template.yaml`) |

## Project Structure

```
cmd/
├── interaction/main.go   # API Gateway → verify, route, enqueue
├── worker/main.go        # SQS → summarize / export → DM
└── scheduler/main.go     # EventBridge cron → send reminders
internal/
├── domain/               # models.go + ports.go (interfaces)
├── app/                  # use-cases
└── platform/            # adapters (dynamodb, sqs, s3, claude, noop, circuit_breaker, rate_limiter…)
template.yaml · samconfig.toml · Makefile · go.mod
```

---

# Setup & Usage

### Prerequisites
- Go 1.21+
- AWS CLI configured (`aws configure`)
- AWS **SAM CLI**
- A Discord application (free)

### 1. Create the Discord application
In the [Discord Developer Portal](https://discord.com/developers/applications): create an app, then grab:
- **Public Key** → `DISCORD_PUBLIC_KEY` (used to verify requests)
- **Bot → Token** → `DISCORD_BOT_TOKEN` (used to DM results)
- **Application ID** (for registering commands / inviting)

### 2. Configure environment
```bash
cp .env.example .env
# fill in:
#   DISCORD_PUBLIC_KEY, DISCORD_BOT_TOKEN
#   CLAUDE_API_KEY   (optional — enables AI summaries)
```
> These become Lambda environment variables at deploy time (via SAM). Without `CLAUDE_API_KEY`, summaries degrade to a no-op message.

### 3. Build & deploy
```bash
make build          # compiles 3 arm64 'bootstrap' binaries
make deploy         # sam deploy --guided  (creates all AWS resources)
```
Copy the **API Gateway URL** from the SAM output.

### 4. Point Discord at your API
In the Developer Portal, set **Interactions Endpoint URL** to your API Gateway URL. Discord sends a `PING`; the interaction Lambda answers it and the URL is validated. ✅

### 5. Register the slash commands
Register the commands (once) with Discord's API — a helper script is included:
```bash
export DISCORD_BOT_TOKEN=... DISCORD_APP_ID=...   # + optional DISCORD_GUILD_ID
go run ./scripts/register
```
> Set `DISCORD_GUILD_ID` (your test server) for **instant** registration; without it, commands register globally (up to ~1h to propagate).

### 6. Invite the bot
Use an OAuth2 URL with the `applications.commands` + `bot` scopes to add it to your server.

### Usage
```
/capture content: Call the vendor about the Q3 invoice
/inbox
/summarize ids: abc123, def456        → "⏳ Summarizing…" then a DM with the summary
/remind content: Deploy the SaaS  in: 2h
/export format: markdown              → "⏳ Generating…" then a DM with the file
/delete id: abc123
```

### Run tests
```bash
make test        # go test -v -race ./...
```

---

## Roadmap / Next Steps
- **PDF / document review** — the `DocumentRepository` + `PDFProcessor` ports are scaffolded; wire a `/review` command (extract text → summarize) to finish it.
- **Live deployment** for a public demo.
