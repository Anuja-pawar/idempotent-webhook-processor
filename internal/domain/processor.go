package domain

import "context"

// WebhookPayload represents the structure of the incoming data we want to process.
type WebhookPayload struct {
	EventID   string                 `json:"event_id"`   // Unique identifier for the event
	ClientID  string                 `json:"client_id"`  // Identifies which business client sent it
	EventType string                 `json:"event_type"` // e.g., "kyc.completed", "user.onboarded"
	Data      map[string]interface{} `json:"data"`       // Flexible metadata payload
}

// IdempotencyRepository defines the contract for checking and storing request states.
type IdempotencyRepository interface {
	// SetLock attempts to acquire an exclusive atomic lock for an event key with a TTL.
	// Returns true if acquired successfully, false if the key already exists.
	SetLock(ctx context.Context, key string) (bool, error)

	// ReleaseLock removes the lock if processing fails early.
	ReleaseLock(ctx context.Context, key string) error
}

// WebhookUseCase defines the contract for handling our core business logic.
type WebhookUseCase interface {
	ProcessWebhook(ctx context.Context, payload *WebhookPayload) error
}
