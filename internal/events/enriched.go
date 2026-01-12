// Package events provides shared event types for the CloudWatch alarm enricher.
package events

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// ViolatingMetric represents a single metric that is currently violating the alarm threshold.
type ViolatingMetric struct {
	Value      float64           `json:"value"`
	Dimensions map[string]string `json:"dimensions"`
	Timestamp  time.Time         `json:"timestamp"`
}

// EnrichedEvent represents a CloudWatch alarm enriched with violating metric details.
// It includes the original alarm state plus specific resources currently violating thresholds.
type EnrichedEvent struct {
	AccountID        string             `json:"accountID"`
	Timestamp        time.Time          `json:"timestamp"`
	Alarm            *types.MetricAlarm `json:"alarm"`
	ViolatingMetrics []ViolatingMetric  `json:"violatingMetrics"`
}
