// Package alarm provides CloudWatch alarm enrichment with detailed metric analysis.
package alarm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm")

// Enricher enriches CloudWatch alarm events with detailed metric analysis.
// It identifies specific resources violating alarm thresholds by querying CloudWatch metrics.
type Enricher interface {
	// Enrich retrieves alarm details and identifies metrics currently violating the threshold.
	// Returns an EnrichedEvent containing the alarm state and violating metrics.
	Enrich(ctx context.Context, alarmName string) (*EnrichedEvent, error)
}

// EnrichedEvent represents a CloudWatch alarm enriched with violating metric details.
// It includes the original alarm state plus specific resources currently violating thresholds.
type EnrichedEvent struct {
	ViolatingMetrics []ViolatingMetric  `json:"violatingMetrics"`
	Timestamp        time.Time          `json:"timestamp"`
	AccountID        string             `json:"accountID"`
	Alarm            *types.MetricAlarm `json:"alarm"`
}

// ViolatingMetric represents a single metric that is currently violating the alarm threshold.
type ViolatingMetric struct {
	Value      float64           `json:"value"`
	Dimensions map[string]string `json:"dimensions"`
	Timestamp  time.Time         `json:"timestamp"`
}

// CloudWatchAPI defines the CloudWatch operations required for alarm enrichment.
type CloudWatchAPI interface {
	DescribeAlarms(
		ctx context.Context,
		input *cloudwatch.DescribeAlarmsInput,
		optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error)

	GetMetricData(
		ctx context.Context,
		input *cloudwatch.GetMetricDataInput,
		optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)

	ListMetrics(
		ctx context.Context,
		input *cloudwatch.ListMetricsInput,
		optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error)
}

// MetricAlarmEnricher implements the Enricher interface for CloudWatch metric alarms.
type MetricAlarmEnricher struct {
	cw     CloudWatchAPI
	logger *slog.Logger
}

// NewMetricAlarmEnricher creates a new MetricAlarmEnricher instance.
func NewMetricAlarmEnricher(
	cw CloudWatchAPI,
	logger *slog.Logger,
) *MetricAlarmEnricher {
	return &MetricAlarmEnricher{
		cw:     cw,
		logger: logger,
	}
}

// Enrich retrieves the alarm details and identifies metrics currently violating the threshold.
// It queries CloudWatch to find the most detailed metrics (those with the most dimensions)
// and determines which specific resources are in violation.
func (e *MetricAlarmEnricher) Enrich(ctx context.Context, alarmName string) (*EnrichedEvent, error) {
	ctx, span := tracer.Start(ctx, "alarm.enrich")
	defer span.End()
	span.SetAttributes(attribute.String("alarm.name", alarmName))

	output, err := e.cw.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmNames: []string{alarmName},
		MaxRecords: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot describe alarm %q: %w", alarmName, err)
	}

	if len(output.MetricAlarms) == 0 {
		return nil, fmt.Errorf("alarm %q not found", alarmName)
	}

	alarm := &output.MetricAlarms[0]

	processed := &EnrichedEvent{
		Alarm:            alarm,
		Timestamp:        time.Now(),
		ViolatingMetrics: []ViolatingMetric{},
	}

	if alarm.StateValue != types.StateValueAlarm {
		e.logger.InfoContext(
			ctx,
			"alarm not in ALARM state; skipping violation analysis",
			slog.String("alarmName", alarmName),
			slog.String("state", string(alarm.StateValue)),
		)
		return processed, nil
	}

	violatingMetrics, err := e.findViolatingMetrics(ctx, alarm)
	if err != nil {
		return nil, fmt.Errorf("cannot find violating metrics for alarm %q: %w", alarmName, err)
	}

	if len(violatingMetrics) == 0 {
		e.logger.WarnContext(
			ctx,
			"alarm in ALARM state but no violations found",
			slog.String("alarmName", alarmName),
			slog.String("namespace", aws.ToString(alarm.Namespace)),
			slog.String("metricName", aws.ToString(alarm.MetricName)),
		)
	}

	processed.ViolatingMetrics = violatingMetrics

	return processed, nil
}

func (e *MetricAlarmEnricher) findViolatingMetrics(ctx context.Context, alarm *types.MetricAlarm) ([]ViolatingMetric, error) {
	dimensionFilters := make([]types.DimensionFilter, 0, len(alarm.Dimensions))
	for _, d := range alarm.Dimensions {
		dimensionFilters = append(dimensionFilters, types.DimensionFilter{
			Name:  d.Name,
			Value: d.Value,
		})
	}

	metricNamespace := aws.ToString(alarm.Namespace)
	metricName := aws.ToString(alarm.MetricName)

	metrics, err := e.findMetricsWithMostDimensions(ctx, metricNamespace, metricName, dimensionFilters)
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return []ViolatingMetric{}, nil
	}

	violating, err := e.analyzeMetricsForViolations(ctx, alarm, metrics)
	if err != nil {
		return nil, err
	}

	return violating, nil
}

func (e *MetricAlarmEnricher) findMetricsWithMostDimensions(
	ctx context.Context,
	namespace, metricName string,
	dimensions []types.DimensionFilter,
) ([]*types.Metric, error) {
	paginator := cloudwatch.NewListMetricsPaginator(e.cw, &cloudwatch.ListMetricsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		Dimensions: dimensions,
	})

	var mostEnriched []*types.Metric
	maxDimensions := len(dimensions)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("cannot list metrics on next page: %w", err)
		}

		for _, m := range page.Metrics {
			dimCount := len(m.Dimensions)

			if dimCount < maxDimensions {
				continue
			}

			// When we find metrics with more dimensions than previously seen,
			// reset the collection since those metrics provide richer detail.
			if dimCount > maxDimensions {
				mostEnriched = []*types.Metric{&m}
				maxDimensions = dimCount
			} else {
				mostEnriched = append(mostEnriched, &m)
			}
		}
	}

	return mostEnriched, nil
}

func (e *MetricAlarmEnricher) analyzeMetricsForViolations(
	ctx context.Context,
	alarm *types.MetricAlarm,
	metrics []*types.Metric,
) ([]ViolatingMetric, error) {
	period := time.Duration(*alarm.Period) * time.Second
	endTime := alignToPeriodBoundary(time.Now(), period)
	evaluationWindow := period * time.Duration(*alarm.EvaluationPeriods)
	startTime := endTime.Add(-evaluationWindow)

	metricQueries := make([]types.MetricDataQuery, len(metrics))
	for i, metric := range metrics {
		metricQueries[i] = types.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m%d", i)),
			MetricStat: &types.MetricStat{
				Metric: metric,
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

		batchViolating, err := e.processBatch(ctx, metricQueries[i:end], metrics[i:end], alarm, startTime, endTime)
		if err != nil {
			return nil, err
		}

		violating = append(violating, batchViolating...)
	}

	return violating, nil
}

func (e *MetricAlarmEnricher) processBatch(
	ctx context.Context,
	queries []types.MetricDataQuery,
	metrics []*types.Metric,
	alarm *types.MetricAlarm,
	startTime, endTime time.Time,
) ([]ViolatingMetric, error) {
	output, err := e.cw.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		MetricDataQueries: queries,
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get metric data: %w", err)
	}

	var violating []ViolatingMetric

	for i, result := range output.MetricDataResults {
		if len(result.Values) == 0 {
			continue
		}

		latestValue := result.Values[len(result.Values)-1]
		timestamp := result.Timestamps[len(result.Timestamps)-1]

		if e.isViolatingThreshold(latestValue, alarm) {
			metric := metrics[i]

			vm := e.createViolatingMetric(*metric, latestValue, timestamp)
			violating = append(violating, vm)
		}
	}

	return violating, nil
}

func (e *MetricAlarmEnricher) isViolatingThreshold(value float64, alarm *types.MetricAlarm) bool {
	threshold := aws.ToFloat64(alarm.Threshold)

	switch alarm.ComparisonOperator {
	case types.ComparisonOperatorGreaterThanThreshold:
		return value > threshold
	case types.ComparisonOperatorGreaterThanOrEqualToThreshold:
		return value >= threshold
	case types.ComparisonOperatorLessThanThreshold:
		return value < threshold
	case types.ComparisonOperatorLessThanOrEqualToThreshold:
		return value <= threshold
	default:
		// For anomaly detection or unknown operators, assume violation
		return true
	}
}

func (e *MetricAlarmEnricher) createViolatingMetric(metric types.Metric, value float64, timestamp time.Time) ViolatingMetric {
	dimensions := make(map[string]string)
	for _, dim := range metric.Dimensions {
		dimensions[aws.ToString(dim.Name)] = aws.ToString(dim.Value)
	}

	return ViolatingMetric{
		Value:      value,
		Dimensions: dimensions,
		Timestamp:  timestamp,
	}
}

// alignToPeriodBoundary aligns a timestamp to CloudWatch period boundaries.
// CloudWatch returns no data for daily metrics when queried with misaligned time windows (e.g., 07:31 to 07:31
// instead of 00:00 to 00:00).
func alignToPeriodBoundary(t time.Time, period time.Duration) time.Time {
	// For 1-day periods, align to midnight UTC
	if period >= 24*time.Hour {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}

	// For shorter periods, round down to the nearest period boundary
	periodSeconds := int64(period.Seconds())
	alignedUnix := (t.Unix() / periodSeconds) * periodSeconds
	return time.Unix(alignedUnix, 0).UTC()
}
