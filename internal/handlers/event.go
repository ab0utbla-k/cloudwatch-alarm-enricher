package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/notifiers"
)

type EventHandler struct {
	enricher alarm.Enricher
	notifier notifiers.Provider
	logger   *slog.Logger
}

func NewEventHandler(enricher alarm.Enricher, notifier notifiers.Provider, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		enricher: enricher,
		notifier: notifier,
		logger:   logger,
	}
}

func (h *EventHandler) HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	enriched, err := h.enricher.Enrich(ctx, event)
	if err != nil {
		return err
	}

	msg := h.BuildNotificationMessaged(enriched)
	err = h.notifier.Send(ctx, aws.ToString(enriched.Alarm.AlarmName), msg)
	return err
}

func (h *EventHandler) BuildNotificationMessaged(event *alarm.EnrichedEvent) string {
	var msg strings.Builder
	a := event.Alarm
	vms := event.ViolatingMetrics

	msg.WriteString(fmt.Sprintf("🚨 CloudWatch Alarm: %s\n", aws.ToString(a.AlarmName)))
	msg.WriteString(fmt.Sprintf("State: %s\n", a.StateValue))
	msg.WriteString(fmt.Sprintf("Reason: %s\n\n", aws.ToString(a.StateReason)))

	if len(event.ViolatingMetrics) == 0 {
		msg.WriteString("No specific services currently violating the threshold.\n")
	} else {
		// Add normal symbols for ops
		msg.WriteString(fmt.Sprintf("Metrics currently violating (%s %.1f) threshold:\n",
			a.ComparisonOperator,
			aws.ToFloat64(a.Threshold)))

		for _, vm := range vms {
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
