// Package dispatch handles sending enriched alarm notifications to various targets.
package dispatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.opentelemetry.io/otel"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/events"
)

var tracer = otel.Tracer("github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/dispatch")

// Sender sends enriched alarm events to a notification target.
type Sender interface {
	// Send dispatches an enriched event to the configured target.
	Send(ctx context.Context, event *events.EnrichedEvent) error
}

// NewSender creates a Sender implementation based on the configured dispatch target.
// Supported targets: sns, eventbridge.
// Returns an error if the dispatch target is unknown.
func NewSender(awsCfg aws.Config, cfg *config.Config) (Sender, error) {
	switch cfg.DispatchTarget {
	case config.TargetSNS:
		client := sns.NewFromConfig(awsCfg)
		return NewSNSSender(client, cfg), nil

	case config.TargetEventBridge:
		client := eventbridge.NewFromConfig(awsCfg)
		return NewEventBridgeSender(client, cfg), nil

	default:
		return nil, fmt.Errorf("unknown dispatch target: %s", cfg.DispatchTarget)
	}
}
