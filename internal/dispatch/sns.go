package dispatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.opentelemetry.io/otel/attribute"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/events"
)

// SNSAPI defines the SNS operations required for sending notifications.
type SNSAPI interface {
	Publish(
		ctx context.Context,
		input *sns.PublishInput,
		optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// SNSSender sends enriched alarms to AWS SNS topics.
type SNSSender struct {
	client SNSAPI
	config *config.Config
}

// NewSNSSender creates a new SNSSender instance.
func NewSNSSender(client SNSAPI, config *config.Config) *SNSSender {
	return &SNSSender{
		client: client,
		config: config,
	}
}

// Send publishes the enriched event to the configured SNS topic.
func (s *SNSSender) Send(ctx context.Context, event *events.EnrichedEvent) error {
	ctx, span := tracer.Start(ctx, "dispatch.send")
	defer span.End()
	span.SetAttributes(
		attribute.String("dispatch.target", string(config.TargetSNS)),
		attribute.String("alarm.name", aws.ToString(event.Alarm.AlarmName)),
	)

	formatter := &TextMessageFormatter{}
	msg, err := formatter.Format(event)
	if err != nil {
		return fmt.Errorf("cannot format message: %w", err)
	}

	input := &sns.PublishInput{
		TopicArn: aws.String(s.config.SNSTopicARN),
		Subject:  aws.String("CloudWatch Alarm - " + aws.ToString(event.Alarm.AlarmName)),
		Message:  aws.String(msg),
	}

	if _, err = s.client.Publish(ctx, input); err != nil {
		return fmt.Errorf("cannot publish to SNS topic %q: %w", s.config.SNSTopicARN, err)
	}

	return nil
}
