package config

import (
	"fmt"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/utils/env"
)

type DispatchTarget string

const (
	TargetSNS         DispatchTarget = "sns"
	TargetEventBridge DispatchTarget = "eventbridge"
	TargetSlack       DispatchTarget = "slack"
	TargetTeams       DispatchTarget = "teams"
)

type Config struct {
	AWSRegion      string
	DispatchTarget DispatchTarget

	EventBusARN     string
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

	dst := env.Get("ALARM_DESTINATION", string(TargetSNS), env.ParseNonEmptyString)
	cfg.DispatchTarget = DispatchTarget(dst)

	switch cfg.DispatchTarget {
	case TargetSNS:
		topicARN, err := env.GetRequired("SNS_TOPIC_ARN", env.ParseNonEmptyString)
		if err != nil {
			return nil, err
		}
		cfg.SNSTopicARN = topicARN
	case TargetEventBridge:
		busARN, err := env.GetRequired("EVENT_BUS_ARN", env.ParseNonEmptyString)
		if err != nil {
			return nil, err
		}
		cfg.EventBusARN = busARN
	case TargetTeams:
		webhookURL, err := env.GetRequired("TEAMS_WEBHOOK_URL", env.ParseNonEmptyString)
		if err != nil {
			return nil, err
		}
		cfg.TeamsWebhookURL = webhookURL
	case TargetSlack:
		webhookURL, err := env.GetRequired("SLACK_WEBHOOK_URL", env.ParseNonEmptyString)
		if err != nil {
			return nil, err
		}
		cfg.SlackWebhookURL = webhookURL
	default:
		return nil, fmt.Errorf("invalid dispatch target: %s", dst)
	}

	return cfg, nil
}
