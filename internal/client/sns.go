package client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type SNSAPI interface {
	Publish(
		ctx context.Context,
		input *sns.PublishInput,
		optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

type SNSClient struct {
	client *sns.Client
}

func (s *SNSClient) Publish(ctx context.Context, input *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return s.client.Publish(ctx, input, optFns...)
}
