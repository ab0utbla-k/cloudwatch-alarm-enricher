package alarm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type Enricher interface {
	Enrich(ctx context.Context, alarmName string) (*EnrichedEvent, error)
}

type EnrichedEvent struct {
	Alarm            *types.MetricAlarm `json:"alarm"`
	ViolatingMetrics []ViolatingMetric  `json:"violatingMetrics"`
	Timestamp        time.Time          `json:"timestamp"`
	Metadata         map[string]string  `json:"metadata"`
}

type ViolatingMetric struct {
	Value      float64           `json:"value"`
	Dimensions map[string]string `json:"dimensions"`
	Timestamp  time.Time         `json:"timestamp"`
}

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

type MetricAlarmEnricher struct {
	cw     CloudWatchAPI
	logger *slog.Logger
}

func NewMetricAlarmEnricher(
	cw CloudWatchAPI,
	logger *slog.Logger,
) *MetricAlarmEnricher {
	return &MetricAlarmEnricher{
		cw:     cw,
		logger: logger,
	}
}

func (e *MetricAlarmEnricher) Enrich(ctx context.Context, alarmName string) (*EnrichedEvent, error) {
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

	metrics, err := e.findEnrichedMetrics(ctx, metricNamespace, metricName, dimensionFilters)
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

func (e *MetricAlarmEnricher) findEnrichedMetrics(
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

			// Found richer level, reset
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
