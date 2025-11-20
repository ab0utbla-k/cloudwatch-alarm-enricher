package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/dispatch"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/handler"
)

func main() {
	startTime := time.Now()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	logger.Info("starting cloudwatch alarm enricher...")

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

	logger.Info(
		"started cloudwatch alarm enricher",
		slog.String("target", string(cfg.DispatchTarget)),
		slog.String("region", cfg.AWSRegion),
		slog.Float64("initDurationSec", time.Since(startTime).Seconds()),
	)

	h := handler.NewEventHandler(enricher, sender, logger)
	lambda.Start(h.HandleRequest)
}
