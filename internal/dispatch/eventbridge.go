package dispatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"go.opentelemetry.io/otel/attribute"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
)

// EventBridgeAPI defines the EventBridge operations required for sending events.
type EventBridgeAPI interface {
	PutEvents(
		ctx context.Context,
		params *eventbridge.PutEventsInput,
		optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

// EventBridgeSender sends enriched alarms to AWS EventBridge.
type EventBridgeSender struct {
	client EventBridgeAPI
	config *config.Config
}

// NewEventBridgeSender creates a new EventBridgeSender instance.
func NewEventBridgeSender(client EventBridgeAPI, config *config.Config) *EventBridgeSender {
	return &EventBridgeSender{
		client: client,
		config: config,
	}
}

// Send publishes the enriched event to the configured EventBridge event bus.
func (s *EventBridgeSender) Send(ctx context.Context, event *alarm.EnrichedEvent) error {
	ctx, span := tracer.Start(ctx, "dispatch.send")
	defer span.End()
	span.SetAttributes(
		attribute.String("dispatch.target", string(config.TargetEventBridge)),
		attribute.String("alarm.name", aws.ToString(event.Alarm.AlarmName)),
	)

	formatter := &JSONMessageFormatter{}
	msg, err := formatter.Format(event)
	if err != nil {
		return fmt.Errorf("cannot format event: %w", err)
	}

	params := &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{{
			Detail:       aws.String(msg),
			DetailType:   aws.String("Alarm Enriched"),
			EventBusName: aws.String(s.config.EventBusARN),
			Source:       aws.String("cloudwatch.alarm.enricher"),
		}},
	}

	if _, err = s.client.PutEvents(ctx, params); err != nil {
		return fmt.Errorf("cannot put events to %q: %w", s.config.EventBusARN, err)
	}

	return nil
}
