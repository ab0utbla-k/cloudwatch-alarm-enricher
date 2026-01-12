// Package notify provides notification senders for the Dispatcher Lambda output.
package notify

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/events"
)

// FormatText converts an enriched event to a human-readable text message.
func FormatText(event *events.EnrichedEvent) (string, error) {
	a := event.Alarm
	var msg strings.Builder

	msg.WriteString("CloudWatch Alarm: ")
	msg.WriteString(aws.ToString(a.AlarmName))
	msg.WriteString("\nState: ")
	msg.WriteString(string(event.Alarm.StateValue))
	msg.WriteString("\nAccountID: ")
	msg.WriteString(event.AccountID)
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

		fmt.Fprintf(&msg, "Metrics currently violating (%s %.1f) threshold:\n",
			symbol,
			aws.ToFloat64(event.Alarm.Threshold))

		for i, vm := range event.ViolatingMetrics {
			dms := make([]string, 0, len(vm.Dimensions))
			for k, v := range vm.Dimensions {
				dms = append(dms, k+"="+v)
			}

			slices.Sort(dms)

			fmt.Fprintf(&msg, "%d. %s, Value: %.2f\t\n",
				i+1,
				strings.Join(dms, ", "),
				vm.Value)
		}
	}

	fmt.Fprintf(&msg, "\nTimestamp: %s", event.Timestamp.Format(time.RFC3339))

	return msg.String(), nil
}

func getComparisonSymbol(op types.ComparisonOperator) (string, error) {
	switch op {
	case types.ComparisonOperatorGreaterThanThreshold:
		return ">", nil
	case types.ComparisonOperatorGreaterThanOrEqualToThreshold:
		return ">=", nil
	case types.ComparisonOperatorLessThanThreshold:
		return "<", nil
	case types.ComparisonOperatorLessThanOrEqualToThreshold:
		return "<=", nil
	default:
		return "", fmt.Errorf("unsupported comparison operator: %s", op)
	}
}
