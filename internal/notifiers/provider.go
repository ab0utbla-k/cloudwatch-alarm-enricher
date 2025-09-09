package notifiers

import "context"

// Provider defines the interface for notification providers
type Provider interface {
	Send(ctx context.Context, subject, message string) error
}
