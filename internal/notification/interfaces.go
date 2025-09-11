package notification

import (
	"context"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
)

// Sender defines the interface for notification providers
type Sender interface {
	Send(ctx context.Context, subject, message string) error
	GetFormat() MessageFormat
}

type MessageBuilder interface {
	BuildSubject(alarmName string) string
	BuildBody(event *alarm.EnrichedEvent) (string, error)
}

type MessageFormat string

const (
	FormatText     MessageFormat = "text"
	FormatMarkdown MessageFormat = "markdown"
	FormatJSON     MessageFormat = "json"
)
