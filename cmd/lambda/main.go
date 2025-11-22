package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/dispatch"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/handler"
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

	var sender dispatch.Sender

	switch cfg.DispatchTarget {
	case config.TargetSNS:
		snsClient := sns.NewFromConfig(awsCfg)
		sender = dispatch.NewSNSSender(snsClient, cfg)
	case config.TargetEventBridge:
		ebClient := eventbridge.NewFromConfig(awsCfg)
		sender = dispatch.NewEventBridgeSender(ebClient, cfg)
	default:
		logger.Error(
			"unknown dispatch target",
			slog.String("target", string(cfg.DispatchTarget)),
		)
		os.Exit(1)
	}

	tp, err := initTracerProvider(ctx)
	if err != nil {
		logger.Error("cannot initialize tracer provider", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
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

func initTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint("localhost:4317"),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create OTLP exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.TelemetrySDKLanguageGo,
			semconv.ServiceNameKey.String(os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create OTEL resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}
