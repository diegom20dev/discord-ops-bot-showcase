package domain

import "time"

type Capture struct {
	ID        string    `dynamodbav:"id" json:"id"`
	UserID    string    `dynamodbav:"user_id" json:"user_id"`
	Content   string    `dynamodbav:"content" json:"content"`
	CreatedAt time.Time `dynamodbav:"created_at" json:"created_at"`
	Summary   string    `dynamodbav:"summary" json:"summary,omitempty"`
	Status    string    `dynamodbav:"status" json:"status"`
}

type TriageEvent struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
}

type Task struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	UserID    string            `json:"user_id"`
	Data      map[string]string `json:"data"`
	Retries   int               `json:"retries,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
}

type Reminder struct {
	ID          string    `dynamodbav:"id" json:"id"`
	UserID      string    `dynamodbav:"user_id" json:"user_id"`
	Content     string    `dynamodbav:"content" json:"content"`
	ScheduledAt time.Time `dynamodbav:"scheduled_at" json:"scheduled_at"`
	CreatedAt   time.Time `dynamodbav:"created_at" json:"created_at"`
	Status      string    `dynamodbav:"status" json:"status"` // pending, sent, dismissed
	TTL         int64     `dynamodbav:"ttl" json:"ttl"`       // Unix timestamp for auto-delete
}

type Export struct {
	ID        string    `dynamodbav:"id" json:"id"`
	UserID    string    `dynamodbav:"user_id" json:"user_id"`
	Format    string    `dynamodbav:"format" json:"format"` // csv, markdown, pdf
	Status    string    `dynamodbav:"status" json:"status"` // pending, completed, failed
	FileURL   string    `dynamodbav:"file_url" json:"file_url,omitempty"`
	CreatedAt time.Time `dynamodbav:"created_at" json:"created_at"`
}

type Document struct {
	ID        string    `dynamodbav:"id" json:"id"`
	UserID    string    `dynamodbav:"user_id" json:"user_id"`
	Filename  string    `dynamodbav:"filename" json:"filename"`
	FileURL   string    `dynamodbav:"file_url" json:"file_url"`
	Content   string    `dynamodbav:"content" json:"content"`       // Texto extraído del PDF
	Summary   string    `dynamodbav:"summary" json:"summary,omitempty"`
	Status    string    `dynamodbav:"status" json:"status"`         // pending, processed, summarized
	CreatedAt time.Time `dynamodbav:"created_at" json:"created_at"`
}
