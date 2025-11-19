package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/client"
)

// Analyzer analyzes metrics to find violations
type Analyzer interface {
	FindViolatingMetrics(ctx context.Context, alarm *types.MetricAlarm) ([]ViolatingMetric, error)
}

type ViolatingMetric struct {
	Value      float64           `json:"value"`
	Dimensions map[string]string `json:"dimensions"`
	Timestamp  time.Time         `json:"timestamp"`
}

type MetricAnalyzer struct {
	cw     client.CloudWatchAPI
	logger *slog.Logger
}

func NewMetricAnalyzer(cw client.CloudWatchAPI, logger *slog.Logger) *MetricAnalyzer {
	return &MetricAnalyzer{
		cw:     cw,
		logger: logger,
	}
}

func (a *MetricAnalyzer) FindViolatingMetrics(ctx context.Context, alarm *types.MetricAlarm) ([]ViolatingMetric, error) {
	dimensionFilters := make([]types.DimensionFilter, 0, len(alarm.Dimensions))
	for _, d := range alarm.Dimensions {
		dimensionFilters = append(dimensionFilters, types.DimensionFilter{
			Name:  d.Name,
			Value: d.Value,
		})
	}

	metricNamespace := aws.ToString(alarm.Namespace)
	metricName := aws.ToString(alarm.MetricName)

	metrics, err := a.findEnrichedMetrics(ctx, metricNamespace, metricName, dimensionFilters)
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		a.logger.InfoContext(
			ctx,
			"no enriched metrics found",
			slog.String("namespace", metricNamespace),
			slog.String("metricName", metricName),
		)

		return nil, nil
	}

	violating, err := a.analyzeMetricsForViolations(ctx, alarm, metrics)
	if err != nil {
		return nil, err
	}

	a.logger.InfoContext(
		ctx,
		"metrics analysis completed",
		slog.Int("totalMetrics", len(metrics)),
		slog.Int("violatingMetrics", len(violating)),
	)

	return violating, nil
}

func (a *MetricAnalyzer) findEnrichedMetrics(
	ctx context.Context,
	namespace, metricName string,
	dimensions []types.DimensionFilter,
) ([]*types.Metric, error) {
	paginator := cloudwatch.NewListMetricsPaginator(a.cw, &cloudwatch.ListMetricsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		Dimensions: dimensions,
	})

	var mostEnriched []*types.Metric
	maxDimensions := len(dimensions)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
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

func (a *MetricAnalyzer) analyzeMetricsForViolations(
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

		batchViolating, err := a.processBatch(ctx, metricQueries[i:end], metrics[i:end], alarm, startTime, endTime)
		if err != nil {
			return nil, err
		}

		violating = append(violating, batchViolating...)
	}

	return violating, nil
}

func (a *MetricAnalyzer) processBatch(
	ctx context.Context,
	queries []types.MetricDataQuery,
	metrics []*types.Metric,
	alarm *types.MetricAlarm,
	startTime, endTime time.Time,
) ([]ViolatingMetric, error) {
	output, err := a.cw.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		MetricDataQueries: queries,
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
	})
	if err != nil {
		return nil, err
	}

	var violating []ViolatingMetric

	for i, result := range output.MetricDataResults {
		if len(result.Values) == 0 {
			continue
		}

		latestValue := result.Values[len(result.Values)-1]
		timestamp := result.Timestamps[len(result.Timestamps)-1]

		if a.isViolatingThreshold(latestValue, alarm) {
			metric := metrics[i]

			vm := a.createViolatingMetric(*metric, latestValue, timestamp)
			violating = append(violating, vm)
		}
	}

	return violating, nil
}

func (a *MetricAnalyzer) isViolatingThreshold(value float64, alarm *types.MetricAlarm) bool {
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

func (a *MetricAnalyzer) createViolatingMetric(metric types.Metric, value float64, timestamp time.Time) ViolatingMetric {
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
