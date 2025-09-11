package notification

import "context"

// Sender defines the interface for notification providers
type Sender interface {
	Send(ctx context.Context, subject, message string) error
}
