package sns

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type Notifier struct {
	client   *sns.Client
	topicArn string
}

func NewNotifier(client *sns.Client, topicArn string) *Notifier {
	return &Notifier{
		client:   client,
		topicArn: topicArn,
	}
}

func (n *Notifier) Send(ctx context.Context, subject, message string) error {
	input := &sns.PublishInput{
		TopicArn: aws.String(n.topicArn),
		Subject:  aws.String(subject),
		Message:  aws.String(message),
	}

	_, err := n.client.Publish(ctx, input)
	return err
}
