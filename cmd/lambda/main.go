package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/dispatch"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/handler"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/telemetry"
)

func main() {
	startTime := time.Now()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	logger.Info("starting cloudwatch alarm enricher")

	cfg, err := config.Load()
	if err != nil {
		logger.Error("cannot load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		logger.Error("cannot load aws config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	cwClient := cloudwatch.NewFromConfig(awsCfg)
	enricher := alarm.NewMetricAlarmEnricher(cwClient, logger)

	sender, err := dispatch.NewSender(awsCfg, cfg)
	if err != nil {
		logger.Error("cannot create sender", slog.String("error", err.Error()))
		os.Exit(1)
	}

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

	logger.Info(
		"started cloudwatch alarm enricher",
		slog.String("target", string(cfg.DispatchTarget)),
		slog.String("region", cfg.AWSRegion),
		slog.Float64("initDurationSec", time.Since(startTime).Seconds()),
	)

	h := handler.NewEventHandler(enricher, sender, logger)
	lambda.Start(
		otellambda.InstrumentHandler(
			h.HandleRequest,
			otellambda.WithTracerProvider(tp),
			otellambda.WithFlusher(tp)),
	)
}
