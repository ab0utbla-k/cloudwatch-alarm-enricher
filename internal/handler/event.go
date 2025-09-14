package handler

import (
	"context"
	"encoding/json"
	"log/slog"

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

	formatter := getMessageFormatter(h.notifier.GetFormat())

	subj := "CloudWatch Alarm - " + aws.ToString(enriched.Alarm.AlarmName)
	msg, err := formatter.Format(enriched)
	if err != nil {
		h.logger.Error("failed to build message", "error", err)
		return err
	}

	err = h.notifier.Send(ctx, subj, msg)
	if err != nil {
		h.logger.Error("failed to send notification",
			"alarm", aws.ToString(enriched.Alarm.AlarmName),
			"error", err)
	}

	return err
}

func getMessageFormatter(format notification.MessageFormat) notification.MessageFormatter {
	switch format {
	case notification.FormatText:
		return &notification.TextMessageFormatter{}
	default:
		return &notification.TextMessageFormatter{}
	}
}
