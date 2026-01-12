// Package publish provides event publishers for the Enricher Lambda output.
package publish

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/events"
)

var tracer = otel.Tracer("github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/publish")

// EventBridgeAPI defines required EventBridge operations.
type EventBridgeAPI interface {
	PutEvents(
		ctx context.Context,
		params *eventbridge.PutEventsInput,
		optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

// Publisher publishes enriched events to EventBridge.
type Publisher struct {
	client       EventBridgeAPI
	eventBusName string
}

// NewPublisher creates a new EventBridge publisher.
func NewPublisher(client EventBridgeAPI, eventBusName string) *Publisher {
	return &Publisher{
		client:       client,
		eventBusName: eventBusName,
	}
}

// Publish sends an enriched event to EventBridge.
func (p *Publisher) Publish(ctx context.Context, event *events.EnrichedEvent) error {
	ctx, span := tracer.Start(ctx, "publish.eventbridge")
	defer span.End()
	span.SetAttributes(
		attribute.String("eventbus.name", p.eventBusName),
		attribute.String("alarm.name", aws.ToString(event.Alarm.AlarmName)),
	)

	detail, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("cannot marshal event: %w", err)
	}

	input := &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{{
			Detail:       aws.String(string(detail)),
			DetailType:   aws.String("Enriched CloudWatch Alarm"),
			EventBusName: aws.String(p.eventBusName),
			Source:       aws.String("cloudwatch.alarm.enricher"),
		}},
	}

	out, err := p.client.PutEvents(ctx, input)
	if err != nil {
		return fmt.Errorf("cannot put event: %w", err)
	}

	if out.FailedEntryCount > 0 {
		entry := out.Entries[0]
		return fmt.Errorf("event rejected: %s - %s",
			aws.ToString(entry.ErrorCode), aws.ToString(entry.ErrorMessage))
	}

	return nil
}
