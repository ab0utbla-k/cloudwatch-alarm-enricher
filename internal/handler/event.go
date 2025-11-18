package handler

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/dispatch"
)

type EventHandler struct {
	enricher alarm.Enricher
	sender   dispatch.Sender
	logger   *slog.Logger
}

func NewEventHandler(enricher alarm.Enricher, sender dispatch.Sender, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		enricher: enricher,
		sender:   sender,
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

	h.logger.InfoContext(
		ctx,
		"processing alarm event",
		slog.String("source", event.Source),
		slog.String("detailType", event.DetailType),
	)

	enriched, err := h.enricher.Enrich(ctx, eventData.AlarmName)
	if err != nil {
		h.logger.ErrorContext(
			ctx,
			"cannot enrich alarm",
			slog.String("alarm", eventData.AlarmName),
			slog.Any("error", err),
		)

		return err
	}

	h.logger.InfoContext(
		ctx,
		"sending notification",
		slog.String("alarm", aws.ToString(enriched.Alarm.AlarmName)),
		slog.Int("violatingCount", len(enriched.ViolatingMetrics)),
	)

	err = h.sender.Send(ctx, enriched)
	if err != nil {
		h.logger.ErrorContext(
			ctx,
			"cannot send notification",
			slog.String("alarm", aws.ToString(enriched.Alarm.AlarmName)),
			slog.Any("error", err),
		)
	}

	return err
}
