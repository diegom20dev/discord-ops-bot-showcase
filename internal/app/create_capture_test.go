package app

import (
	"context"
	"testing"

	"github.com/diegom20dev/discord-ops-bot/internal/domain"
)

// Mock implementations for testing
type mockRepository struct {
	saved []*domain.Capture
}

func (m *mockRepository) SaveCapture(ctx context.Context, capture *domain.Capture) error {
	m.saved = append(m.saved, capture)
	return nil
}

func (m *mockRepository) GetCapture(ctx context.Context, id string) (*domain.Capture, error) {
	for _, c := range m.saved {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) ListCapturesByUser(ctx context.Context, userID string) ([]*domain.Capture, error) {
	var captures []*domain.Capture
	for _, c := range m.saved {
		if c.UserID == userID {
			captures = append(captures, c)
		}
	}
	return captures, nil
}

func (m *mockRepository) UpdateCapture(ctx context.Context, capture *domain.Capture) error {
	return nil
}

type mockQueue struct {
	tasks []*domain.Task
}

func (m *mockQueue) EnqueueTask(ctx context.Context, task *domain.Task) error {
	m.tasks = append(m.tasks, task)
	return nil
}

// Tests
func TestCreateCapture(t *testing.T) {
	repo := &mockRepository{}
	queue := &mockQueue{}
	uc := NewCreateCaptureUC(repo, queue)

	ctx := context.Background()
	capture, err := uc.Execute(ctx, CreateCaptureInput{
		UserID:  "user123",
		Content: "Test content",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capture.UserID != "user123" {
		t.Errorf("Expected userID=user123, got %s", capture.UserID)
	}

	if capture.Content != "Test content" {
		t.Errorf("Expected content='Test content', got %s", capture.Content)
	}

	if capture.Status != "pending" {
		t.Errorf("Expected status=pending, got %s", capture.Status)
	}

	if len(repo.saved) != 1 {
		t.Errorf("Expected 1 saved capture, got %d", len(repo.saved))
	}

	if len(queue.tasks) != 1 {
		t.Errorf("Expected 1 queued task, got %d", len(queue.tasks))
	}

	if queue.tasks[0].Type != "summarize" {
		t.Errorf("Expected task type=summarize, got %s", queue.tasks[0].Type)
	}
}
