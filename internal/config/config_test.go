package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_SNSTarget(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("ALARM_DESTINATION", "sns")
	t.Setenv("SNS_TOPIC_ARN", "arn:aws:sns:us-east-1:123456789012:test-topic")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "us-east-1", cfg.AWSRegion)
	assert.Equal(t, TargetSNS, cfg.DispatchTarget)
	assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:test-topic", cfg.SNSTopicARN)
	assert.Empty(t, cfg.EventBusARN)
	assert.Empty(t, cfg.SlackWebhookURL)
	assert.Empty(t, cfg.TeamsWebhookURL)
}

func TestLoad_SNSTargetDefault(t *testing.T) {
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("SNS_TOPIC_ARN", "arn:aws:sns:us-west-2:123456789012:topic")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, TargetSNS, cfg.DispatchTarget)
	assert.Equal(t, "arn:aws:sns:us-west-2:123456789012:topic", cfg.SNSTopicARN)
}

func TestLoad_EventBridgeTarget(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("ALARM_DESTINATION", "eventbridge")
	t.Setenv("EVENT_BUS_ARN", "arn:aws:events:eu-west-1:123456789012:event-bus/test-bus")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "eu-west-1", cfg.AWSRegion)
	assert.Equal(t, TargetEventBridge, cfg.DispatchTarget)
	assert.Equal(t, "arn:aws:events:eu-west-1:123456789012:event-bus/test-bus", cfg.EventBusARN)
	assert.Empty(t, cfg.SNSTopicARN)
}

func TestLoad_SlackTarget(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("ALARM_DESTINATION", "slack")
	t.Setenv("SLACK_WEBHOOK_URL", "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, TargetSlack, cfg.DispatchTarget)
	assert.Equal(t, "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX", cfg.SlackWebhookURL)
}

func TestLoad_TeamsTarget(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("ALARM_DESTINATION", "teams")
	t.Setenv("TEAMS_WEBHOOK_URL", "https://outlook.office.com/webhook/test")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, TargetTeams, cfg.DispatchTarget)
	assert.Equal(t, "https://outlook.office.com/webhook/test", cfg.TeamsWebhookURL)
}

func TestLoad_MissingAWSRegion(t *testing.T) {
	t.Setenv("ALARM_DESTINATION", "sns")
	t.Setenv("SNS_TOPIC_ARN", "arn:aws:sns:us-east-1:123456789012:topic")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "AWS_REGION")
}

func TestLoad_MissingSNSTopicARN(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("ALARM_DESTINATION", "sns")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "SNS_TOPIC_ARN")
}

func TestLoad_MissingEventBusARN(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("ALARM_DESTINATION", "eventbridge")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "EVENT_BUS_ARN")
}

func TestLoad_InvalidDispatchTarget(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("ALARM_DESTINATION", "invalid-target")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid dispatch target")
}
