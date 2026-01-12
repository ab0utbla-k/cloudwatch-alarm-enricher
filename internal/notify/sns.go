package notify

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/events"
)

var tracer = otel.Tracer("github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/notify")

// SNSAPI defines required SNS operations.
type SNSAPI interface {
	Publish(
		ctx context.Context,
		input *sns.PublishInput,
		optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// SNS sends notifications to SNS.
type SNS struct {
	client   SNSAPI
	topicARN string
}

// NewSNS creates a new SNS sender.
func NewSNS(client SNSAPI, topicARN string) *SNS {
	return &SNS{
		client:   client,
		topicARN: topicARN,
	}
}

// Send publishes an enriched event to SNS.
func (s *SNS) Send(ctx context.Context, event *events.EnrichedEvent) error {
	ctx, span := tracer.Start(ctx, "notify.sns")
	defer span.End()
	span.SetAttributes(
		attribute.String("sns.topic_arn", s.topicARN),
		attribute.String("alarm.name", aws.ToString(event.Alarm.AlarmName)),
	)

	msg, err := FormatText(event)
	if err != nil {
		return fmt.Errorf("cannot format message: %w", err)
	}

	input := &sns.PublishInput{
		TopicArn: aws.String(s.topicARN),
		Subject:  aws.String("CloudWatch Alarm - " + aws.ToString(event.Alarm.AlarmName)),
		Message:  aws.String(msg),
	}

	if _, err = s.client.Publish(ctx, input); err != nil {
		return fmt.Errorf("cannot publish to SNS: %w", err)
	}

	return nil
}
