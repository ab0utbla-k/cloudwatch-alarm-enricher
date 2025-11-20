package handler

import (
	"context"
	"encoding/json"
	"errors"
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
		h.logger.ErrorContext(
			ctx,
			"cannot parse event detail",
			slog.String("error", err.Error()),
		)
		return err
	}

	if eventData.AlarmName == "" {
		err := errors.New("alarm name is empty")
		h.logger.ErrorContext(
			ctx,
			"invalid event",
			slog.String("error", err.Error()),
		)
		return err
	}

	enriched, err := h.enricher.Enrich(ctx, eventData.AlarmName)
	if err != nil {
		h.logger.ErrorContext(
			ctx,
			"cannot enrich alarm",
			slog.String("alarmName", eventData.AlarmName),
			slog.String("error", err.Error()),
		)
		return err
	}

	enriched.AccountID = event.AccountID

	if err := h.sender.Send(ctx, enriched); err != nil {
		h.logger.ErrorContext(
			ctx,
			"cannot send notification",
			slog.String("alarmName", aws.ToString(enriched.Alarm.AlarmName)),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}
