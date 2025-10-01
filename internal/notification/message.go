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

type TextMessageFormatter struct{}

func (f *TextMessageFormatter) Format(event *alarm.EnrichedEvent) (string, error) {
	a := event.Alarm
	var msg strings.Builder

	msg.WriteString("🚨 CloudWatch Alarm: ")
	msg.WriteString(aws.ToString(a.AlarmName))
	msg.WriteString("\nState: ")
	msg.WriteString(string(event.Alarm.StateValue))
	msg.WriteString("\nReason: ")
	msg.WriteString(aws.ToString(a.StateReason))
	msg.WriteString("\n\n")

	if len(event.ViolatingMetrics) == 0 {
		msg.WriteString("No specific services currently violating the threshold.\n")
	} else {
		symbol, err := getComparisonSymbol(event.Alarm.ComparisonOperator)
		if err != nil {
			return "", err
		}

		_, err = fmt.Fprintf(&msg, "Metrics currently violating (%s %.1f) threshold:\n",
			symbol,
			aws.ToFloat64(event.Alarm.Threshold))
		if err != nil {
			return "", err
		}

		for _, vm := range event.ViolatingMetrics {
			dms := make([]string, 0, len(vm.Dimensions))
			for k, v := range vm.Dimensions {
				dms = append(dms, k+"="+v)
			}

			_, err = fmt.Fprintf(&msg, "• %s, Value: %.2f",
				strings.Join(dms, ", "),
				vm.Value)
			if err != nil {
				return "", err
			}

			msg.WriteByte('\n')
		}
	}

	_, err := fmt.Fprintf(&msg, "\nTimestamp: %s", event.Timestamp.Format(time.RFC3339))
	if err != nil {
		return "", err
	}

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
