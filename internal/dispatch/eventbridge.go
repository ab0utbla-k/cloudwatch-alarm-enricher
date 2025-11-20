package dispatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
)

type EventBridgeSender struct {
	client EventBridgeAPI
	config *config.Config
}

func NewEventBridgeSender(client EventBridgeAPI, config *config.Config) *EventBridgeSender {
	return &EventBridgeSender{
		client: client,
		config: config,
	}
}

func (s *EventBridgeSender) Send(ctx context.Context, event *alarm.EnrichedEvent) error {
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
