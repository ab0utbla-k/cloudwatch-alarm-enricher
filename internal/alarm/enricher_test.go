package alarm

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupEnricher(t *testing.T) (*CloudWatchAPIMock, *MetricAlarmEnricher) {
	t.Helper()

	mockCW := new(CloudWatchAPIMock)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return mockCW, NewMetricAlarmEnricher(mockCW, logger)
}

func newDescribeAlarmInput(alarmName string) *cloudwatch.DescribeAlarmsInput {
	return &cloudwatch.DescribeAlarmsInput{
		AlarmNames: []string{alarmName},
		MaxRecords: aws.Int32(1),
	}
}

func newMetricAlarm(alarmName, metricName, namespace string, state types.StateValue) types.MetricAlarm {
	return types.MetricAlarm{
		AlarmName:          aws.String(alarmName),
		StateValue:         state,
		MetricName:         aws.String(metricName),
		Namespace:          aws.String(namespace),
		Statistic:          types.StatisticAverage,
		Period:             aws.Int32(60),
		EvaluationPeriods:  aws.Int32(1),
		Threshold:          aws.Float64(50.0),
		ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
	}
}

func newDimension(name, value string) types.Dimension {
	return types.Dimension{
		Name:  aws.String(name),
		Value: aws.String(value),
	}
}

func newMetric(metricName, namespace string, dimensions []types.Dimension) types.Metric {
	return types.Metric{
		MetricName: aws.String(metricName),
		Namespace:  aws.String(namespace),
		Dimensions: dimensions,
	}
}

func newMetricDataResult(id string, values []float64, timestamps []time.Time) types.MetricDataResult {
	return types.MetricDataResult{
		Id:         aws.String(id),
		Values:     values,
		Timestamps: timestamps,
		StatusCode: types.StatusCodeComplete,
	}
}

func TestEnrich_AlarmNotFound(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "nonexistent-alarm"

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{},
	}, nil).Once()

	_, err := enricher.Enrich(context.Background(), alarmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	mockCW.AssertExpectations(t)
}

func TestEnrich_DescribeAlarmsError(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "test-alarm"
	expectedError := errors.New("describe alarms failed")

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return((*cloudwatch.DescribeAlarmsOutput)(nil), expectedError).Once()

	_, err := enricher.Enrich(context.Background(), alarmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), expectedError.Error())
	mockCW.AssertExpectations(t)
}

func TestEnrich_AlarmNotInAlarmState(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "ok-alarm"

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{
			newMetricAlarm(alarmName, "", "", types.StateValueOk),
		},
	}, nil).Once()

	event, err := enricher.Enrich(context.Background(), alarmName)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Empty(t, event.ViolatingMetrics)
	assert.Equal(t, types.StateValueOk, event.Alarm.StateValue)
	mockCW.AssertExpectations(t)
}

func TestEnrich_ListMetricsError(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "test-alarm-with-list-metrics-error"
	expectedError := errors.New("list metrics failed")

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{
			{
				AlarmName:          aws.String(alarmName),
				StateValue:         types.StateValueAlarm,
				MetricName:         aws.String("TestMetric"),
				Namespace:          aws.String("TestNamespace"),
				Statistic:          types.StatisticAverage,
				Period:             aws.Int32(300),
				EvaluationPeriods:  aws.Int32(1),
				Threshold:          aws.Float64(10.0),
				ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
			},
		},
	}, nil).Once()

	mockCW.On("ListMetrics",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.ListMetricsInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return((*cloudwatch.ListMetricsOutput)(nil), expectedError).Once()

	_, err := enricher.Enrich(context.Background(), alarmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), expectedError.Error())
	mockCW.AssertExpectations(t)
}

func TestEnrich_NoEnrichedMetricsFound(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "test-alarm-no-enriched-metrics"

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{
			{
				AlarmName:          aws.String(alarmName),
				StateValue:         types.StateValueAlarm,
				MetricName:         aws.String("TestMetric"),
				Namespace:          aws.String("TestNamespace"),
				Statistic:          types.StatisticAverage,
				Period:             aws.Int32(300),
				EvaluationPeriods:  aws.Int32(1),
				Threshold:          aws.Float64(10.0),
				ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("InstanceId"),
						Value: aws.String("i-12345"),
					},
				},
			},
		},
	}, nil).Once()

	// Mock ListMetrics to return an empty list
	mockCW.On("ListMetrics",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.ListMetricsInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.ListMetricsOutput{
		Metrics: []types.Metric{},
	}, nil).Once()

	event, err := enricher.Enrich(context.Background(), alarmName)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Empty(t, event.ViolatingMetrics)
	mockCW.AssertExpectations(t)
}

func TestEnrich_GetMetricDataError(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "test-alarm-with-get-metric-data-error"
	expectedError := errors.New("get metric data failed")

	alarm := newMetricAlarm(alarmName, "TestMetric", "TestNamespace", types.StateValueAlarm)
	alarm.Dimensions = []types.Dimension{newDimension("InstanceId", "i-12345")}

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{alarm},
	}, nil).Once()

	mockCW.On("ListMetrics",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.ListMetricsInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.ListMetricsOutput{
		Metrics: []types.Metric{
			newMetric("TestMetric", "TestNamespace", []types.Dimension{newDimension("InstanceId", "i-12345")}),
		},
	}, nil).Once()

	mockCW.On("GetMetricData",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.GetMetricDataInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return((*cloudwatch.GetMetricDataOutput)(nil), expectedError).Once()

	_, err := enricher.Enrich(context.Background(), alarmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), expectedError.Error())
	mockCW.AssertExpectations(t)
}

func TestEnrich_SuccessfulEnrichmentWithViolations(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "test-alarm-with-violations"
	metricName := "CPUUtilization"
	namespace := "AWS/EC2"
	instanceID := "i-0abcdef1234567890"

	alarm := newMetricAlarm(alarmName, metricName, namespace, types.StateValueAlarm)
	alarm.Dimensions = []types.Dimension{newDimension("InstanceId", instanceID)}

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{alarm},
	}, nil).Once()

	mockCW.On("ListMetrics",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.ListMetricsInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.ListMetricsOutput{
		Metrics: []types.Metric{
			newMetric(metricName, namespace, []types.Dimension{newDimension("InstanceId", instanceID)}),
		},
	}, nil).Once()

	mockCW.On("GetMetricData",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.GetMetricDataInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{
			newMetricDataResult("m0", []float64{40.0, 60.0}, []time.Time{
				time.Now().Add(-2 * time.Minute),
				time.Now().Add(-1 * time.Minute),
			}),
		},
	}, nil).Once()

	event, err := enricher.Enrich(context.Background(), alarmName)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.NotEmpty(t, event.ViolatingMetrics)
	assert.Len(t, event.ViolatingMetrics, 1)
	assert.Equal(t, 60.0, event.ViolatingMetrics[0].Value)
	assert.Equal(t, instanceID, event.ViolatingMetrics[0].Dimensions["InstanceId"])
	mockCW.AssertExpectations(t)
}

func TestEnrich_SuccessfulEnrichmentWithoutViolations(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "test-alarm-no-violations"
	metricName := "CPUUtilization"
	namespace := "AWS/EC2"
	instanceID := "i-0abcdef1234567890"

	alarm := newMetricAlarm(alarmName, metricName, namespace, types.StateValueAlarm)
	alarm.Dimensions = []types.Dimension{newDimension("InstanceId", instanceID)}

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{alarm},
	}, nil).Once()

	mockCW.On("ListMetrics",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.ListMetricsInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.ListMetricsOutput{
		Metrics: []types.Metric{
			newMetric(metricName, namespace, []types.Dimension{newDimension("InstanceId", instanceID)}),
		},
	}, nil).Once()

	mockCW.On("GetMetricData",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.GetMetricDataInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{
			newMetricDataResult("m0", []float64{40.0, 30.0}, []time.Time{
				time.Now().Add(-2 * time.Minute),
				time.Now().Add(-1 * time.Minute),
			}),
		},
	}, nil).Once()

	event, err := enricher.Enrich(context.Background(), alarmName)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Empty(t, event.ViolatingMetrics)
	mockCW.AssertExpectations(t)
}

func TestEnrich_AlarmWithNoDimensions(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "test-alarm-no-dimensions"
	metricName := "AccountLimit"
	namespace := "AWS/Usage"

	alarm := newMetricAlarm(alarmName, metricName, namespace, types.StateValueAlarm)
	alarm.Threshold = aws.Float64(80.0)
	alarm.Period = aws.Int32(300)
	alarm.Dimensions = []types.Dimension{}

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{alarm},
	}, nil).Once()

	mockCW.On("ListMetrics",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.ListMetricsInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.ListMetricsOutput{
		Metrics: []types.Metric{
			newMetric(metricName, namespace, []types.Dimension{}),
		},
	}, nil).Once()

	mockCW.On("GetMetricData",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.GetMetricDataInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{
			newMetricDataResult("m0", []float64{85.0}, []time.Time{time.Now().Add(-5 * time.Minute)}),
		},
	}, nil).Once()

	event, err := enricher.Enrich(context.Background(), alarmName)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.NotEmpty(t, event.ViolatingMetrics)
	assert.Len(t, event.ViolatingMetrics, 1)
	assert.Equal(t, 85.0, event.ViolatingMetrics[0].Value)
	assert.Empty(t, event.ViolatingMetrics[0].Dimensions)
	mockCW.AssertExpectations(t)
}

func TestEnrich_MetricDataWithEmptyValues(t *testing.T) {
	mockCW, enricher := setupEnricher(t)
	alarmName := "test-alarm-empty-values"
	metricName := "RequestCount"
	namespace := "AWS/ELB"
	lbName := "my-load-balancer"

	alarm := newMetricAlarm(alarmName, metricName, namespace, types.StateValueAlarm)
	alarm.Statistic = types.StatisticSum
	alarm.Threshold = aws.Float64(1000.0)
	alarm.Dimensions = []types.Dimension{newDimension("LoadBalancerName", lbName)}

	mockCW.On("DescribeAlarms",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		newDescribeAlarmInput(alarmName),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{alarm},
	}, nil).Once()

	mockCW.On("ListMetrics",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.ListMetricsInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.ListMetricsOutput{
		Metrics: []types.Metric{
			newMetric(metricName, namespace, []types.Dimension{newDimension("LoadBalancerName", lbName)}),
		},
	}, nil).Once()

	mockCW.On("GetMetricData",
		mock.MatchedBy(func(ctx context.Context) bool { return ctx != nil }),
		mock.AnythingOfType("*cloudwatch.GetMetricDataInput"),
		mock.AnythingOfType("[]func(*cloudwatch.Options)"),
	).Return(&cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{
			newMetricDataResult("m0", []float64{}, []time.Time{}),
		},
	}, nil).Once()

	event, err := enricher.Enrich(context.Background(), alarmName)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Empty(t, event.ViolatingMetrics)
	mockCW.AssertExpectations(t)
}

func TestAlignToPeriodBoundary(t *testing.T) {
	// 1-day period aligns to midnight UTC
	// This is critical: CloudWatch returns NO DATA for daily metrics with misaligned time windows
	input := time.Date(2025, 10, 2, 6, 38, 15, 0, time.UTC)
	period := 24 * time.Hour
	expected := time.Date(2025, 10, 2, 0, 0, 0, 0, time.UTC)
	result := alignToPeriodBoundary(input, period)
	assert.Equal(t, expected, result)
	assert.NotEqual(t, input, result, "Time must be aligned, not left as-is")

	// 1-day period at midnight stays at midnight
	input = time.Date(2025, 10, 2, 0, 0, 0, 0, time.UTC)
	period = 24 * time.Hour
	expected = time.Date(2025, 10, 2, 0, 0, 0, 0, time.UTC)
	result = alignToPeriodBoundary(input, period)
	assert.Equal(t, expected, result)

	// 5-minute period rounds down
	input = time.Date(2025, 10, 2, 6, 38, 15, 0, time.UTC)
	period = 5 * time.Minute
	expected = time.Date(2025, 10, 2, 6, 35, 0, 0, time.UTC)
	result = alignToPeriodBoundary(input, period)
	assert.Equal(t, expected, result)

	// 5-minute period at exact boundary
	input = time.Date(2025, 10, 2, 6, 35, 0, 0, time.UTC)
	period = 5 * time.Minute
	expected = time.Date(2025, 10, 2, 6, 35, 0, 0, time.UTC)
	result = alignToPeriodBoundary(input, period)
	assert.Equal(t, expected, result)

	// 1-hour period rounds down
	input = time.Date(2025, 10, 2, 6, 38, 15, 0, time.UTC)
	period = 1 * time.Hour
	expected = time.Date(2025, 10, 2, 6, 0, 0, 0, time.UTC)
	result = alignToPeriodBoundary(input, period)
	assert.Equal(t, expected, result)

	// 1-minute period rounds down
	input = time.Date(2025, 10, 2, 6, 38, 42, 0, time.UTC)
	period = 1 * time.Minute
	expected = time.Date(2025, 10, 2, 6, 38, 0, 0, time.UTC)
	result = alignToPeriodBoundary(input, period)
	assert.Equal(t, expected, result)
}
