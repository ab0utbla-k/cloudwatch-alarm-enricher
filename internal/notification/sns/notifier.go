package sns

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/client"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/notification"
)

type Notifier struct {
	client client.SNSAPI
	config *config.Config
}

func New(client client.SNSAPI, config *config.Config) *Notifier {
	return &Notifier{
		client: client,
		config: config,
	}
}

func (n *Notifier) Send(ctx context.Context, subject, message string) error {
	input := &sns.PublishInput{
		TopicArn: aws.String(n.config.SNSTopicARN),
		Subject:  aws.String(subject),
		Message:  aws.String(message),
	}

	_, err := n.client.Publish(ctx, input)
	return err
}

func (n *Notifier) GetFormat() notification.MessageFormat {
	return notification.FormatText
}
