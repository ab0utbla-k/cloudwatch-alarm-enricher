package client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// SNSAPI defines the interface for SNS operations.
type SNSAPI interface {
	Publish(
		ctx context.Context,
		input *sns.PublishInput,
		optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// SNSClient wraps the AWS SNS client and implements SNSAPI interface.
type SNSClient struct {
	client *sns.Client
}

func NewSNS(client *sns.Client) *SNSClient {
	return &SNSClient{
		client: client,
	}
}

// Publish publishes a message to an SNS topic.
func (s *SNSClient) Publish(ctx context.Context, input *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return s.client.Publish(ctx, input, optFns...)
}
