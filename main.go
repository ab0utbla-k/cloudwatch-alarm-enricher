package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type LambdaHandler struct {
	cwClient    *cloudwatch.Client
	snsClient   *sns.Client
	snsTopicArn string
	logger      *slog.Logger
}

func NewLambdaHandler() (*LambdaHandler, error) {
	snsTopicArn := os.Getenv("SNS_TOPIC_ARN")
	if snsTopicArn == "" {
		return nil, fmt.Errorf("SNS_TOPIC_ARN environment variable is required")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &LambdaHandler{
		cwClient:    cloudwatch.NewFromConfig(cfg),
		snsClient:   sns.NewFromConfig(cfg),
		snsTopicArn: snsTopicArn,
		logger:      logger,
	}, nil
}

type ViolatingMetric struct {
	Value      float64
	Dimensions map[string]string
}

func main() {
	handler, err := NewLambdaHandler()
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	lambda.Start(handler.HandleRequest)
}

func (h *LambdaHandler) HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	h.logger.Info("Processing alarm event", slog.Any("event", event))

	var eventData struct {
		AlarmName string `json:"alarmName"`
	}
	if err := json.Unmarshal(event.Detail, &eventData); err != nil {
		return err
	}

	out, err := h.cwClient.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmNames: []string{eventData.AlarmName},
	})
	if err != nil {
		return err
	}

	alarm := out.MetricAlarms[0]
	alarmName := aws.ToString(alarm.AlarmName)

	if alarm.StateValue != types.StateValueAlarm {
		h.logger.Info("Alarm resolved during processing, sending brief notification",
			"state", alarm.StateValue)

		return nil
	}

	metricNamespace := aws.ToString(alarm.Namespace)
	metricName := aws.ToString(alarm.MetricName)

	var dimensionFilters []types.DimensionFilter
	for _, d := range alarm.Dimensions {
		dimensionFilters = append(dimensionFilters, types.DimensionFilter{
			Name:  d.Name,
			Value: d.Value,
		})
	}

	h.logger.Info("Processing alarm",
		"alarm", alarmName,
		"namespace", metricNamespace,
		"metric", metricName)

	metrics, err := h.filterMetrics(ctx, metricNamespace, metricName, dimensionFilters)
	if err != nil {
		return err
	}

	violatingMetrics, err := h.findViolatingMetrics(ctx, &alarm, metrics)
	if err != nil {
		return err
	}

	message := h.buildNotificationMessage(&alarm, violatingMetrics)
	if err = h.sendNotification(ctx, alarmName, message); err != nil {
		return err
	}

	h.logger.Info("Successfully sent enriched notification",
		"alarm", alarmName,
		"topic", h.snsTopicArn)

	return nil
}

func (h *LambdaHandler) findViolatingMetrics(
	ctx context.Context,
	alarm *types.MetricAlarm,
	metrics []*types.Metric,
) ([]ViolatingMetric, error) {
	if len(metrics) == 0 {
		return nil, nil
	}

	endTime := time.Now()
	evaluationWindow := time.Duration(*alarm.Period**alarm.EvaluationPeriods) * time.Second
	startTime := endTime.Add(-evaluationWindow)

	metricQueries := make([]types.MetricDataQuery, len(metrics))
	for i, m := range metrics {
		metricQueries[i] = types.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m%d", i)),
			MetricStat: &types.MetricStat{
				Metric: m,
				Period: alarm.Period,
				Stat:   aws.String(string(alarm.Statistic)),
			},
			ReturnData: aws.Bool(true),
		}
	}

	var violating []ViolatingMetric
	const batchSize = 500

	for i := 0; i < len(metricQueries); i += batchSize {
		end := i + batchSize
		if end > len(metricQueries) {
			end = len(metricQueries)
		}

		output, err := h.cwClient.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
			MetricDataQueries: metricQueries[i:end],
			StartTime:         aws.Time(startTime),
			EndTime:           aws.Time(endTime),
		})
		if err != nil {
			return nil, err
		}

		for j, result := range output.MetricDataResults {
			values := result.Values
			if len(values) == 0 {
				continue
			}

			latest := values[len(values)-1]
			threshold := aws.ToFloat64(alarm.Threshold)

			var violatesThreshold bool
			switch alarm.ComparisonOperator {
			case types.ComparisonOperatorGreaterThanThreshold:
				violatesThreshold = latest > threshold
			case types.ComparisonOperatorGreaterThanOrEqualToThreshold:
				violatesThreshold = latest >= threshold
			case types.ComparisonOperatorLessThanThreshold:
				violatesThreshold = latest < threshold
			case types.ComparisonOperatorLessThanOrEqualToThreshold:
				violatesThreshold = latest <= threshold
			default:
				// For anomaly detection or unknown operators, assume alarm firing means violation
				violatesThreshold = true
			}

			if violatesThreshold {
				metric := metricQueries[i+j].MetricStat.Metric
				vm := h.extractMetricDimensions(metric, latest)
				violating = append(violating, *vm)
			}
		}
	}

	return violating, nil
}

func (h *LambdaHandler) filterMetrics(
	ctx context.Context,
	namespace, metricName string,
	dimensions []types.DimensionFilter,
) ([]*types.Metric, error) {
	paginator := cloudwatch.NewListMetricsPaginator(h.cwClient, &cloudwatch.ListMetricsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		Dimensions: dimensions,
	})

	var bestMetrics []*types.Metric
	maxDimensions := len(dimensions)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, m := range page.Metrics {
			dimCount := len(m.Dimensions)

			if dimCount <= len(dimensions) {
				continue
			}

			// Found better enrichment level, reset collection
			if dimCount > maxDimensions {
				maxDimensions = dimCount
				bestMetrics = []*types.Metric{&m}
			} else if dimCount == maxDimensions {
				bestMetrics = append(bestMetrics, &m)
			}
		}
	}

	return bestMetrics, nil
}

func (h *LambdaHandler) extractMetricDimensions(metric *types.Metric, value float64) *ViolatingMetric {
	dimensions := make(map[string]string)
	for _, dim := range metric.Dimensions {
		dimensions[aws.ToString(dim.Name)] = aws.ToString(dim.Value)
	}
	return &ViolatingMetric{
		Value:      value,
		Dimensions: dimensions,
	}
}

func (h *LambdaHandler) buildNotificationMessage(alarm *types.MetricAlarm, violatingMetrics []ViolatingMetric) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("🚨 CloudWatch Alarm: %s\n", aws.ToString(alarm.AlarmName)))
	msg.WriteString(fmt.Sprintf("State: %s\n", alarm.StateValue))
	msg.WriteString(fmt.Sprintf("Reason: %s\n\n", aws.ToString(alarm.StateReason)))

	if len(violatingMetrics) == 0 {
		msg.WriteString("No specific services currently violating the threshold.\n")
	} else {
		msg.WriteString(fmt.Sprintf("Metrics currently violating %s %.1f threshold:\n",
			alarm.ComparisonOperator,
			aws.ToFloat64(alarm.Threshold)))

		for _, vm := range violatingMetrics {
			var dims []string
			for k, v := range vm.Dimensions {
				dims = append(dims, fmt.Sprintf("%s=%s", k, v))
			}
			msg.WriteString(fmt.Sprintf("• %s, Value: %.2f\n", strings.Join(dims, ", "), vm.Value))
		}
	}

	msg.WriteString(fmt.Sprintf("\nTimestamp: %s", time.Now().Format(time.RFC3339)))

	return msg.String()
}

func (h *LambdaHandler) sendNotification(ctx context.Context, alarmName, message string) error {
	subject := fmt.Sprintf("CloudWatch Alarm - %s", alarmName)

	input := &sns.PublishInput{
		TopicArn: aws.String(h.snsTopicArn),
		Subject:  aws.String(subject),
		Message:  aws.String(message),
	}

	_, err := h.snsClient.Publish(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Implement proper anomaly detection alarm handling
