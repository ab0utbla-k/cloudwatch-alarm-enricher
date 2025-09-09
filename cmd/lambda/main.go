package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/client"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/handlers"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/metrics"
	snsn "github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/notifiers/sns"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	snsTopicArn := os.Getenv("SNS_TOPIC_ARN")
	if snsTopicArn == "" {
		slog.Error("SNS_TOPIC_ARN environment variable is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("failed to load aws config")
		os.Exit(1)
	}

	snsClient := sns.NewFromConfig(cfg)
	cwClient := client.NewClient(cloudwatch.NewFromConfig(cfg))

	analyzer := metrics.NewMetricAnalyzer(cwClient, logger)
	notifier := snsn.NewNotifier(snsClient, snsTopicArn)
	enricher := alarm.NewMetricEnricher(cwClient, analyzer, logger)

	h := handlers.NewEventHandler(enricher, notifier, logger)
	lambda.Start(h.HandleRequest)
}
