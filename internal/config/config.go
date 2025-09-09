package config

import (
	"fmt"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/utils/env"
)

type NotificationProvider string

const (
	ProviderSNS   NotificationProvider = "sns"
	ProviderSlack NotificationProvider = "slack"
	ProviderTeams NotificationProvider = "teams"
)

type Config struct {
	AWSRegion            string
	NotificationProvider NotificationProvider

	SNSTopicARN     string
	SlackWebhookURL string
	TeamsWebhookURL string
}

func Load() (*Config, error) {
	cfg := &Config{}

	region, err := env.GetRequired("AWS_REGION", env.ParseNonEmptyString)
	if err != nil {
		return nil, err
	}

	cfg.AWSRegion = region

	provider := env.Get("NOTIFICATION_PROVIDER", string(ProviderSNS), env.ParseNonEmptyString)

	switch NotificationProvider(provider) {
	case ProviderSNS:
		topicARN, err := env.GetRequired("SNS_TOPIC_ARN", env.ParseNonEmptyString)
		if err != nil {
			return nil, err
		}
		cfg.SNSTopicARN = topicARN
	default:
		return nil, fmt.Errorf("invalid notification provider: %s", provider)
	}

	return cfg, nil
}
