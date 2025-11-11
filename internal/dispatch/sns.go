package dispatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/client"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
)

type SNSSender struct {
	client client.SNSAPI
	config *config.Config
}

func NewSNSSender(client client.SNSAPI, config *config.Config) *SNSSender {
	return &SNSSender{
		client: client,
		config: config,
	}
}

func (s *SNSSender) Send(ctx context.Context, event *alarm.EnrichedEvent) error {
	formatter := &TextMessageFormatter{}
	msg, err := formatter.Format(event)
	if err != nil {
		return err
	}

	input := &sns.PublishInput{
		TopicArn: aws.String(s.config.SNSTopicARN),
		Subject:  aws.String("CloudWatch Alarm - " + aws.ToString(event.Alarm.AlarmName)),
		Message:  aws.String(msg),
	}

	_, err = s.client.Publish(ctx, input)
	return err
}
