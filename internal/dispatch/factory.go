package dispatch

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
)

// NewSender creates and returns the appropriate Sender implementation
// based on the dispatch target specified in the configuration.
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
