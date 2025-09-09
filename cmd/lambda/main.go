package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/client"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/config"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/handler"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/metrics"
	snsnotifier "github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/notification/sns"
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

	snsClient := sns.NewFromConfig(awsCfg)
	cwClient := client.NewCloudWatch(cloudwatch.NewFromConfig(awsCfg))

	analyzer := metrics.NewMetricAnalyzer(cwClient, logger)
	notifier := snsnotifier.New(snsClient, cfg)
	enricher := alarm.NewMetricEnricher(cwClient, analyzer, logger)

	h := handler.NewEventHandler(enricher, notifier, logger)
	lambda.Start(h.HandleRequest)
}
