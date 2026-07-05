package domain

import "context"

// Repository port para persistencia de capturas
type CaptureRepository interface {
	SaveCapture(ctx context.Context, capture *Capture) error
	GetCapture(ctx context.Context, id string) (*Capture, error)
	ListCapturesByUser(ctx context.Context, userID string) ([]*Capture, error)
	UpdateCapture(ctx context.Context, capture *Capture) error
}

// Queue port para encolamiento de tareas async
type Queue interface {
	EnqueueTask(ctx context.Context, task *Task) error
}

// Discord client port para interacciones
type DiscordClient interface {
	ReplyToInteraction(ctx context.Context, interactionID, interactionToken, message string) error
	SendFollowUp(ctx context.Context, userID, message string) error
}

// Summarizer port para resumir capturas
type Summarizer interface {
	Summarize(ctx context.Context, content string) (string, error)
}

// RateLimiter port para limitar acciones por usuario
type RateLimiter interface {
	AllowAction(ctx context.Context, userID, action string) (bool, error)
}

// ReminderRepository port para persistencia de recordatorios
type ReminderRepository interface {
	SaveReminder(ctx context.Context, reminder *Reminder) error
	GetReminder(ctx context.Context, id string) (*Reminder, error)
	ListRemindersByUser(ctx context.Context, userID string) ([]*Reminder, error)
	ListPendingReminders(ctx context.Context) ([]*Reminder, error)
	UpdateReminder(ctx context.Context, reminder *Reminder) error
}

// ExportRepository port para persistencia de exports
type ExportRepository interface {
	SaveExport(ctx context.Context, export *Export) error
	GetExport(ctx context.Context, id string) (*Export, error)
	UpdateExport(ctx context.Context, export *Export) error
}

// FileStorage port para almacenar archivos
type FileStorage interface {
	Upload(ctx context.Context, fileName string, data []byte) (string, error) // Returns file URL
}

// DocumentRepository port para persistencia de documentos
type DocumentRepository interface {
	SaveDocument(ctx context.Context, doc *Document) error
	GetDocument(ctx context.Context, id string) (*Document, error)
	ListDocumentsByUser(ctx context.Context, userID string) ([]*Document, error)
	UpdateDocument(ctx context.Context, doc *Document) error
}

// PDFProcessor port para procesar PDFs
type PDFProcessor interface {
	ExtractText(ctx context.Context, pdfURL string) (string, error)
}
