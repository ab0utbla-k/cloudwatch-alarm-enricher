package dispatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
)

type SNSSender struct {
	client SNSAPI
	config *config.Config
}

func NewSNSSender(client SNSAPI, config *config.Config) *SNSSender {
	return &SNSSender{
		client: client,
		config: config,
	}
}

func (s *SNSSender) Send(ctx context.Context, event *alarm.EnrichedEvent) error {
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
