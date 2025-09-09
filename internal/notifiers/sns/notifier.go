package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/metrics"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// Provider defines the interface for notification providers
type Provider interface {
	Send(ctx context.Context, subject, message string) error
}

type SNSProvider struct {
	client   *sns.Client
	topicArn string
}

func NewSNSProvider(client *sns.Client, topicArn string) *SNSProvider {
	return &SNSProvider{
		client:   client,
		topicArn: topicArn,
	}
}

func (p *SNSProvider) Send(ctx context.Context, subject, message string) error {
	input := &sns.PublishInput{
		TopicArn: aws.String(p.topicArn),
		Subject:  aws.String(subject),
		Message:  aws.String(message),
	}

	_, err := p.client.Publish(ctx, input)
	return err
}

func (p *SNSProvider) buildNotificationMessage(alarm *types.MetricAlarm, violatingMetrics []metrics.ViolatingMetric) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("🚨 CloudWatch Alarm: %s\n", aws.ToString(alarm.AlarmName)))
	msg.WriteString(fmt.Sprintf("State: %s\n", alarm.StateValue))
	msg.WriteString(fmt.Sprintf("Reason: %s\n\n", aws.ToString(alarm.StateReason)))

	if len(violatingMetrics) == 0 {
		msg.WriteString("No specific services currently violating the threshold.\n")
	} else {
		msg.WriteString(fmt.Sprintf("Metrics currently violating %s %.1f threshold:\n",
			alarm.ComparisonOperator,
			aws.ToFloat64(alarm.Threshold)))

		for _, vm := range violatingMetrics {
			var dims []string
			for k, v := range vm.Dimensions {
				dims = append(dims, fmt.Sprintf("%s=%s", k, v))
			}
			msg.WriteString(fmt.Sprintf("• %s, Value: %.2f\n", strings.Join(dims, ", "), vm.Value))
		}
	}

	msg.WriteString(fmt.Sprintf("\nTimestamp: %s", time.Now().Format(time.RFC3339)))

	return msg.String()
}

//type SlackProvider struct {
//	webhookURL string
//	channel    string
//}
//
//func NewSlack(webhookURL, channel string) Provider {
//	return &SlackProvider{
//		webhookURL: webhookURL,
//		channel:    channel,
//	}
//}
//
//func (s *SlackProvider) Send(ctx context.Context, subject, message string) error {
//	// Implementation for Slack notifications would go here
//	return nil
//}
//
//type TeamsProvider struct {
//	webhookURL string
//}
//
//func NewTeams(webhookURL string) Provider {
//	return &TeamsProvider{
//		webhookURL: webhookURL,
//	}
//}
//
//func (t *TeamsProvider) Send(ctx context.Context, subject, message string) error {
//	// Implementation for Teams notifications would go here
//	return nil
//}
