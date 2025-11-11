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
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		logger.Error("failed to load aws config", "error", err)
		os.Exit(1)
	}

	cwClient := cloudwatch.NewFromConfig(awsCfg)

	analyzer := metrics.NewMetricAnalyzer(cwClient, logger)
	enricher := alarm.NewMetricEnricher(cwClient, analyzer, logger)

	var sender dispatch.Sender
	switch cfg.DispatchTarget {
	case config.TargetSNS:
		snsClient := sns.NewFromConfig(awsCfg)
		sender = dispatch.NewSNSSender(snsClient, cfg)
	case config.TargetEventBridge:
		ebClient := eventbridge.NewFromConfig(awsCfg)
		sender = dispatch.NewEventBridgeSender(ebClient, cfg)
	}

	h := handler.NewEventHandler(enricher, sender, logger)
	lambda.Start(h.HandleRequest)
}
