package dispatch

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ab0utbla-k/cloudwatch-alarm-enricher/internal/alarm"
)

func TestTextMessageFormatter_NoViolations(t *testing.T) {
	formatter := &TextMessageFormatter{}

	event := &alarm.EnrichedEvent{
		Alarm: &types.MetricAlarm{
			AlarmName:   aws.String("TestAlarm"),
			StateValue:  types.StateValueAlarm,
			StateReason: aws.String("Threshold breached"),
			Threshold:   aws.Float64(100.0),
		},
		ViolatingMetrics: []alarm.ViolatingMetric{},
		Timestamp:        time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC),
	}

	message, err := formatter.Format(event)
	require.NoError(t, err)
	assert.Contains(t, message, "TestAlarm")
	assert.Contains(t, message, "ALARM")
	assert.Contains(t, message, "No specific services currently violating the threshold")
}

func TestTextMessageFormatter_WithViolations(t *testing.T) {
	formatter := &TextMessageFormatter{}

	event := &alarm.EnrichedEvent{
		Alarm: &types.MetricAlarm{
			AlarmName:          aws.String("HighErrorRate"),
			StateValue:         types.StateValueAlarm,
			StateReason:        aws.String("Threshold breached"),
			Threshold:          aws.Float64(10.0),
			ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
		},
		ViolatingMetrics: []alarm.ViolatingMetric{
			{
				Value: 15.5,
				Dimensions: map[string]string{
					"ServiceName": "api-service",
					"Environment": "production",
				},
				Timestamp: time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC),
			},
			{
				Value: 20.3,
				Dimensions: map[string]string{
					"ServiceName": "worker-service",
				},
				Timestamp: time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC),
			},
		},
		Timestamp: time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC),
	}

	message, err := formatter.Format(event)
	require.NoError(t, err)
	require.Contains(t, message, "HighErrorRate")
	require.Contains(t, message, "ALARM")
	require.Contains(t, message, "> 10.0")
	require.Contains(t, message, "api-service")
	require.Contains(t, message, "15.50")
	require.Contains(t, message, "worker-service")
	require.Contains(t, message, "20.30")
}

func TestTextMessageFormatter_DimensionsSorted(t *testing.T) {
	formatter := &TextMessageFormatter{}

	event := &alarm.EnrichedEvent{
		Alarm: &types.MetricAlarm{
			AlarmName:          aws.String("TestAlarm"),
			StateValue:         types.StateValueAlarm,
			StateReason:        aws.String("Test"),
			Threshold:          aws.Float64(5.0),
			ComparisonOperator: types.ComparisonOperatorGreaterThanOrEqualToThreshold,
		},
		ViolatingMetrics: []alarm.ViolatingMetric{
			{
				Value: 10.0,
				Dimensions: map[string]string{
					"ZZZ": "last",
					"AAA": "first",
					"MMM": "middle",
				},
				Timestamp: time.Now(),
			},
		},
		Timestamp: time.Now(),
	}

	message, err := formatter.Format(event)
	require.NoError(t, err)
	require.Contains(t, message, "AAA=first, MMM=middle, ZZZ=last")
}

func TestJSONMessageFormatter_NoViolations(t *testing.T) {
	formatter := &JSONMessageFormatter{}

	event := &alarm.EnrichedEvent{
		Alarm: &types.MetricAlarm{
			AlarmName:   aws.String("TestAlarm"),
			StateValue:  types.StateValueAlarm,
			StateReason: aws.String("Threshold breached"),
			Threshold:   aws.Float64(100.0),
		},
		ViolatingMetrics: []alarm.ViolatingMetric{},
		Timestamp:        time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC),
	}

	message, err := formatter.Format(event)
	require.NoError(t, err)
	require.NotEmpty(t, message)

	var result alarm.EnrichedEvent
	err = json.Unmarshal([]byte(message), &result)
	require.NoError(t, err)

	assert.Equal(t, "TestAlarm", aws.ToString(result.Alarm.AlarmName))
	assert.Equal(t, types.StateValueAlarm, result.Alarm.StateValue)
	assert.Empty(t, result.ViolatingMetrics)
}

func TestJSONMessageFormatter_WithViolations(t *testing.T) {
	formatter := &JSONMessageFormatter{}

	timestamp := time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC)
	event := &alarm.EnrichedEvent{
		Alarm: &types.MetricAlarm{
			AlarmName:          aws.String("HighErrorRate"),
			StateValue:         types.StateValueAlarm,
			StateReason:        aws.String("Threshold breached"),
			Threshold:          aws.Float64(10.0),
			ComparisonOperator: types.ComparisonOperatorGreaterThanThreshold,
		},
		ViolatingMetrics: []alarm.ViolatingMetric{
			{
				Value: 15.5,
				Dimensions: map[string]string{
					"ServiceName": "api-service",
					"Environment": "production",
				},
				Timestamp: timestamp,
			},
			{
				Value: 20.3,
				Dimensions: map[string]string{
					"ServiceName": "worker-service",
				},
				Timestamp: timestamp,
			},
		},
		Timestamp: timestamp,
	}

	message, err := formatter.Format(event)
	require.NoError(t, err)
	require.NotEmpty(t, message)

	var result alarm.EnrichedEvent
	err = json.Unmarshal([]byte(message), &result)
	require.NoError(t, err)

	assert.Equal(t, "HighErrorRate", aws.ToString(result.Alarm.AlarmName))
	assert.Equal(t, types.StateValueAlarm, result.Alarm.StateValue)
	assert.Equal(t, 10.0, aws.ToFloat64(result.Alarm.Threshold))

	require.Len(t, result.ViolatingMetrics, 2)
	assert.Equal(t, 15.5, result.ViolatingMetrics[0].Value)
	assert.Equal(t, "api-service", result.ViolatingMetrics[0].Dimensions["ServiceName"])
	assert.Equal(t, "production", result.ViolatingMetrics[0].Dimensions["Environment"])
	assert.Equal(t, 20.3, result.ViolatingMetrics[1].Value)
	assert.Equal(t, "worker-service", result.ViolatingMetrics[1].Dimensions["ServiceName"])
}

func TestGetComparisonSymbol(t *testing.T) {
	symbol, err := getComparisonSymbol(types.ComparisonOperatorGreaterThanThreshold)
	require.NoError(t, err)
	require.Equal(t, ">", symbol)

	symbol, err = getComparisonSymbol(types.ComparisonOperatorGreaterThanOrEqualToThreshold)
	require.NoError(t, err)
	require.Equal(t, ">=", symbol)

	symbol, err = getComparisonSymbol(types.ComparisonOperatorLessThanThreshold)
	require.NoError(t, err)
	require.Equal(t, "<", symbol)

	symbol, err = getComparisonSymbol(types.ComparisonOperatorLessThanOrEqualToThreshold)
	require.NoError(t, err)
	require.Equal(t, "<=", symbol)
}
