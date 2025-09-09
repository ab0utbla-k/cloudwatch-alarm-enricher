package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/client"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/metrics"
)

type Enricher interface {
	Enrich(ctx context.Context, event events.CloudWatchEvent) (*EnrichedEvent, error)
}

// EnrichedEvent represents the processed alarm event
type EnrichedEvent struct {
	Alarm            *types.MetricAlarm        `json:"alarm"`
	ViolatingMetrics []metrics.ViolatingMetric `json:"violatingMetrics"`
	Timestamp        time.Time                 `json:"timestamp"`
	Metadata         map[string]string         `json:"metadata"`
}

type MetricEnricher struct {
	cw       client.CloudWatch
	analyzer metrics.Analyzer
	logger   *slog.Logger
}

func NewMetricEnricher(
	cw client.CloudWatch,
	analyzer metrics.Analyzer,
	logger *slog.Logger,
) *MetricEnricher {
	return &MetricEnricher{
		cw:       cw,
		analyzer: analyzer,
		logger:   logger,
	}
}

func (e *MetricEnricher) Enrich(ctx context.Context, event events.CloudWatchEvent) (*EnrichedEvent, error) {
	e.logger.Info("processing alarm event",
		"source", event.Source,
		"detailType", event.DetailType,
	)

	var eventData struct {
		AlarmName string `json:"alarmName"`
	}

	if err := json.Unmarshal(event.Detail, &eventData); err != nil {
		return nil, err
	}

	if eventData.AlarmName == "" {
		return nil, errors.New("alarm name not found in event")
	}

	output, err := e.cw.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmNames: []string{eventData.AlarmName},
		MaxRecords: aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}

	if len(output.MetricAlarms) == 0 {
		return nil, fmt.Errorf("alarm not found: %s", eventData.AlarmName)
	}

	alarm := &output.MetricAlarms[0]

	e.logger.Info("processing alarm",
		"alarm", aws.ToString(alarm.AlarmName),
		"state", alarm.StateValue,
		"metric", aws.ToString(alarm.MetricName),
		"namespace", aws.ToString(alarm.Namespace))

	processed := &EnrichedEvent{
		Alarm:            alarm,
		Timestamp:        time.Now(),
		ViolatingMetrics: []metrics.ViolatingMetric{},
	}

	if alarm.StateValue != types.StateValueAlarm {
		e.logger.Info("alarm resolved during processing",
			"state", alarm.StateValue)

		processed.Metadata = map[string]string{"status": "resolved"}

		return processed, nil
	}

	violatingMetrics, err := e.analyzer.FindViolatingMetrics(ctx, alarm)
	if err != nil {
		return nil, err
	}

	processed.ViolatingMetrics = violatingMetrics

	return processed, nil
}
