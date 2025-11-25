// Package config handles application configuration loading from environment variables.
package config

import (
	"fmt"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/utils/env"
)

// DispatchTarget represents the notification destination for enriched alarms.
type DispatchTarget string

const (
	// TargetSNS sends notifications to AWS SNS.
	TargetSNS DispatchTarget = "sns"
	// TargetEventBridge sends events to AWS EventBridge.
	TargetEventBridge DispatchTarget = "eventbridge"
	// TargetSlack sends notifications to Slack (not yet implemented).
	TargetSlack DispatchTarget = "slack"
	// TargetTeams sends notifications to Microsoft Teams (not yet implemented).
	TargetTeams DispatchTarget = "teams"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	AWSRegion      string
	DispatchTarget DispatchTarget

	EventBusARN     string
	SNSTopicARN     string
	SlackWebhookURL string
	TeamsWebhookURL string
}

// Load reads and validates configuration from environment variables.
// Returns an error if required variables are missing or invalid.
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
