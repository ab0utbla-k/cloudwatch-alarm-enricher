package dispatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.opentelemetry.io/otel"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
)

var tracer = otel.Tracer("github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/dispatch")

type EventBridgeAPI interface {
	PutEvents(
		ctx context.Context,
		params *eventbridge.PutEventsInput,
		optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type Sender interface {
	Send(ctx context.Context, event *alarm.EnrichedEvent) error
	// Send(ctx context.Context, subject, message string) error
	// GetFormat() MessageFormat
}

type MessageFormatter interface {
	Format(event *alarm.EnrichedEvent) (string, error)
}

type SNSAPI interface {
	Publish(
		ctx context.Context,
		input *sns.PublishInput,
		optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// type MessageFormat string
//
// const (
//	FormatJSON     MessageFormat = "json"
//	FormatText     MessageFormat = "text"
//	FormatMarkdown MessageFormat = "markdown"
// )
