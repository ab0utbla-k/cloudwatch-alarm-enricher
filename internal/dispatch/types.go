package dispatch

import (
	"context"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
)

type Sender interface {
	Send(ctx context.Context, event *alarm.EnrichedEvent) error
	// Send(ctx context.Context, subject, message string) error
	// GetFormat() MessageFormat
}

type MessageFormatter interface {
	Format(event *alarm.EnrichedEvent) (string, error)
}

// type MessageFormat string
//
// const (
//	FormatJSON     MessageFormat = "json"
//	FormatText     MessageFormat = "text"
//	FormatMarkdown MessageFormat = "markdown"
// )
