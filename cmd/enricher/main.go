package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/env"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/publish"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	eventBusName := env.Get("EVENT_BUS_NAME", "default", env.ParseNonEmptyString)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("cannot load aws config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	enricher := alarm.NewMetricAlarmEnricher(cloudwatch.NewFromConfig(awsCfg), logger)
	publisher := publish.NewPublisher(eventbridge.NewFromConfig(awsCfg), eventBusName)

	tp, err := telemetry.NewTracerProvider(ctx)
	if err != nil {
		logger.Error("cannot initialize tracer provider", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := tp.Shutdown(shutdownCtx); err != nil {
			logger.Error("cannot shutdown tracer provider", slog.String("error", err.Error()))
		}
	}()

	logger.Info("started enricher", slog.String("eventBus", eventBusName))

	handler := func(ctx context.Context, event events.CloudWatchEvent) error {
		return handleRequest(ctx, event, enricher, publisher, logger)
	}

	lambda.Start(
		otellambda.InstrumentHandler(
			handler,
			otellambda.WithTracerProvider(tp),
			otellambda.WithFlusher(tp)),
	)
}

func handleRequest(
	ctx context.Context,
	event events.CloudWatchEvent,
	enricher alarm.Enricher,
	publisher *publish.Publisher,
	logger *slog.Logger,
) error {
	var detail struct {
		AlarmName string `json:"alarmName"`
	}

	if err := json.Unmarshal(event.Detail, &detail); err != nil {
		logger.ErrorContext(ctx, "cannot parse event detail", slog.String("error", err.Error()))
		return err
	}

	if detail.AlarmName == "" {
		err := errors.New("alarm name is empty")
		logger.ErrorContext(ctx, "invalid event", slog.String("error", err.Error()))
		return err
	}

	enriched, err := enricher.Enrich(ctx, detail.AlarmName)
	if err != nil {
		logger.ErrorContext(ctx, "cannot enrich alarm",
			slog.String("alarmName", detail.AlarmName),
			slog.String("error", err.Error()))
		return err
	}

	enriched.AccountID = event.AccountID

	if err := publisher.Publish(ctx, enriched); err != nil {
		logger.ErrorContext(ctx, "cannot publish enriched event",
			slog.String("alarmName", aws.ToString(enriched.Alarm.AlarmName)),
			slog.String("error", err.Error()))
		return err
	}

	logger.InfoContext(ctx, "enriched event published",
		slog.String("alarmName", aws.ToString(enriched.Alarm.AlarmName)))

	return nil
}
