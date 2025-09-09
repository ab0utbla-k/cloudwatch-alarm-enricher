package awsapi

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

// CloudWatchAPI defines the interface for CloudWatch operations.
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

// Client wraps the AWS CloudWatch client and implements CloudWatchAPI interface.
type Client struct {
	cw *cloudwatch.Client
}

// NewClient creates a new CloudWatch service instance.
func NewClient(cw *cloudwatch.Client) *Client {
	return &Client{
		cw: cw,
	}
}

// DescribeAlarms retrieves information about CloudWatch alarms.
func (c *Client) DescribeAlarms(ctx context.Context, input *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	return c.cw.DescribeAlarms(ctx, input, optFns...)
}

// GetMetricData retrieves metric data for CloudWatch metrics.
func (c *Client) GetMetricData(ctx context.Context, input *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	return c.cw.GetMetricData(ctx, input, optFns...)
}

// ListMetrics retrieves a list of CloudWatch metrics.
func (c *Client) ListMetrics(ctx context.Context, input *cloudwatch.ListMetricsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error) {
	return c.cw.ListMetrics(ctx, input, optFns...)
}
