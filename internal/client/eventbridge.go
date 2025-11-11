package client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
)

type EventBridgeAPI interface {
	PutEvents(
		ctx context.Context,
		params *eventbridge.PutEventsInput,
		optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type EventBridgeClient struct {
	client *eventbridge.Client
}

func (eb *EventBridgeClient) PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	return eb.client.PutEvents(ctx, params, optFns...)
}
