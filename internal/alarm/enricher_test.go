package alarm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
