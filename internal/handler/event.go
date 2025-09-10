package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/notification"
)

type EventHandler struct {
	enricher alarm.Enricher
	notifier notification.Sender
	logger   *slog.Logger
}

func NewEventHandler(enricher alarm.Enricher, notifier notification.Sender, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		enricher: enricher,
		notifier: notifier,
		logger:   logger,
	}
}

func (h *EventHandler) HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	var eventData struct {
		AlarmName string `json:"alarmName"`
	}

	if err := json.Unmarshal(event.Detail, &eventData); err != nil {
		return err
	}

	h.logger.Info("alarm event received",
		"source", event.Source,
		"detailType", event.DetailType,
	)

	enriched, err := h.enricher.Enrich(ctx, eventData.AlarmName)
	if err != nil {
		h.logger.Error("failed to enrich alarm",
			"alarm", eventData.AlarmName,
			"error", err)
		return err
	}

	h.logger.Info("sending notification",
		"alarm", aws.ToString(enriched.Alarm.AlarmName),
		"violating_count", len(enriched.ViolatingMetrics))

	msg := h.BuildNotificationMessage(enriched)

	err = h.notifier.Send(ctx, aws.ToString(enriched.Alarm.AlarmName), msg)
	if err != nil {
		h.logger.Error("failed to send notification",
			"alarm", aws.ToString(enriched.Alarm.AlarmName),
			"error", err)
	}

	return err
}

func (h *EventHandler) BuildNotificationMessage(event *alarm.EnrichedEvent) string {
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
