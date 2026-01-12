package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	lambdaevents "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/env"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/events"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/notify"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	topicARN, err := env.GetRequired("SNS_TOPIC_ARN", env.ParseNonEmptyString)
	if err != nil {
		logger.Error("cannot load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("cannot load aws config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	sender := notify.NewSNS(sns.NewFromConfig(awsCfg), topicARN)

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

	logger.Info("started dispatcher", slog.String("topicARN", topicARN))

	handler := func(ctx context.Context, event lambdaevents.CloudWatchEvent) error {
		return handleRequest(ctx, event, sender, logger)
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
	event lambdaevents.CloudWatchEvent,
	sender *notify.SNS,
	logger *slog.Logger,
) error {
	var enriched events.EnrichedEvent
	if err := json.Unmarshal(event.Detail, &enriched); err != nil {
		logger.ErrorContext(ctx, "cannot parse enriched event", slog.String("error", err.Error()))
		return err
	}

	if err := sender.Send(ctx, &enriched); err != nil {
		logger.ErrorContext(ctx, "cannot send notification",
			slog.String("alarmName", aws.ToString(enriched.Alarm.AlarmName)),
			slog.String("error", err.Error()))
		return err
	}

	logger.InfoContext(ctx, "notification sent",
		slog.String("alarmName", aws.ToString(enriched.Alarm.AlarmName)))

	return nil
}
