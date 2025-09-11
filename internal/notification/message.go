package notification

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
)

type comparisonSymbol string

const (
	GreaterThanOrEqualToThreshold comparisonSymbol = ">="
	GreaterThanThreshold          comparisonSymbol = ">"
	LessThanThreshold             comparisonSymbol = "<"
	LessThanOrEqualToThreshold    comparisonSymbol = "<="
)

type TextMessageBuilder struct{}

func (b *TextMessageBuilder) BuildSubject(alarmName string) string {
	return fmt.Sprintf("CloudWatch Alarm - %s", alarmName)
}

func (b *TextMessageBuilder) BuildBody(event *alarm.EnrichedEvent) (string, error) {
	a := event.Alarm
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("🚨 CloudWatch Alarm: %s\n", aws.ToString(a.AlarmName)))
	msg.WriteString(fmt.Sprintf("State: %s\n", event.Alarm.StateValue))
	msg.WriteString(fmt.Sprintf("Reason: %s\n\n", aws.ToString(a.StateReason)))

	if len(event.ViolatingMetrics) == 0 {
		msg.WriteString("No specific services currently violating the threshold.\n")
	} else {
		symbol, err := getComparisonSymbol(event.Alarm.ComparisonOperator)
		if err != nil {
			return "", err
		}

		msg.WriteString(fmt.Sprintf(
			"Metrics currently violating (%s %.1f) threshold:\n",
			symbol,
			aws.ToFloat64(event.Alarm.Threshold)))

		for _, vm := range event.ViolatingMetrics {
			dims := make([]string, 0, len(vm.Dimensions))
			for k, v := range vm.Dimensions {
				dims = append(dims, fmt.Sprintf("%s=%s", k, v))
			}
			msg.WriteString(fmt.Sprintf(
				"• %s, Value: %.2f\n",
				strings.Join(dims, ", "),
				vm.Value))
		}
	}

	msg.WriteString(fmt.Sprintf(
		"\nTimestamp: %s",
		event.Timestamp.Format(time.RFC3339)))

	return msg.String(), nil
}

func getComparisonSymbol(op types.ComparisonOperator) (string, error) {
	switch op {
	case types.ComparisonOperatorGreaterThanThreshold:
		return string(GreaterThanThreshold), nil
	case types.ComparisonOperatorGreaterThanOrEqualToThreshold:
		return string(GreaterThanOrEqualToThreshold), nil
	case types.ComparisonOperatorLessThanThreshold:
		return string(LessThanThreshold), nil
	case types.ComparisonOperatorLessThanOrEqualToThreshold:
		return string(LessThanOrEqualToThreshold), nil
	default:
		return "", fmt.Errorf("unsupported comparison operator: %s", op)
	}
}
