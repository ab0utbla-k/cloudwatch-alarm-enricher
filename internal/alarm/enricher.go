package alarm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/client"
	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/metrics"
)

type Enricher interface {
	Enrich(ctx context.Context, alarmName string) (*EnrichedEvent, error)
}

// EnrichedEvent represents the processed alarm event
type EnrichedEvent struct {
	Alarm            *types.MetricAlarm        `json:"alarm"`
	ViolatingMetrics []metrics.ViolatingMetric `json:"violatingMetrics"`
	Timestamp        time.Time                 `json:"timestamp"`
	Metadata         map[string]string         `json:"metadata"`
}

type MetricEnricher struct {
	cw       client.CloudWatchAPI
	analyzer metrics.Analyzer
	logger   *slog.Logger
}

func NewMetricEnricher(
	cw client.CloudWatchAPI,
	analyzer metrics.Analyzer,
	logger *slog.Logger,
) *MetricEnricher {
	return &MetricEnricher{
		cw:       cw,
		analyzer: analyzer,
		logger:   logger,
	}
}

func (e *MetricEnricher) Enrich(ctx context.Context, alarmName string) (*EnrichedEvent, error) {
	if alarmName == "" {
		return nil, errors.New("alarm name not found in event")
	}

	output, err := e.cw.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmNames: []string{alarmName},
		MaxRecords: aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}

	if len(output.MetricAlarms) == 0 {
		return nil, fmt.Errorf("alarm not found: %s", alarmName)
	}

	alarm := &output.MetricAlarms[0]

	processed := &EnrichedEvent{
		Alarm:            alarm,
		Timestamp:        time.Now(),
		ViolatingMetrics: []metrics.ViolatingMetric{},
	}

	if alarm.StateValue != types.StateValueAlarm {
		e.logger.InfoContext(ctx,
			"alarm resolved during processing",
			slog.String("alarm", alarmName),
			slog.String("state", string(alarm.StateValue)),
		)

		processed.Metadata = map[string]string{"status": "resolved"}

		return processed, nil
	}

	violatingMetrics, err := e.analyzer.FindViolatingMetrics(ctx, alarm)
	if err != nil {
		return nil, err
	}

	if len(violatingMetrics) == 0 {
		e.logger.WarnContext(
			ctx,
			"alarm in ALARM state but no violations found",
			slog.String("alarm", alarmName),
			slog.String("metric", aws.ToString(alarm.MetricName)),
			slog.String("namespace", aws.ToString(alarm.Namespace)),
		)
	}

	processed.ViolatingMetrics = violatingMetrics

	return processed, nil
}
