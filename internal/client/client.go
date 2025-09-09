package client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

// CloudWatch defines the interface for CloudWatch operations.
type CloudWatch interface {
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

// Client wraps the AWS CloudWatch client and implements CloudWatch interface.
type Client struct {
	cloudwatch *cloudwatch.Client
}

// NewClient creates a new CloudWatch service instance.
func NewClient(cwClient *cloudwatch.Client) *Client {
	return &Client{
		cloudwatch: cwClient,
	}
}

// DescribeAlarms retrieves information about CloudWatch alarms.
func (c *Client) DescribeAlarms(ctx context.Context, input *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	return c.cloudwatch.DescribeAlarms(ctx, input, optFns...)
}

// GetMetricData retrieves metric data for CloudWatch metrics.
func (c *Client) GetMetricData(ctx context.Context, input *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	return c.cloudwatch.GetMetricData(ctx, input, optFns...)
}

// ListMetrics retrieves a list of CloudWatch metrics.
func (c *Client) ListMetrics(ctx context.Context, input *cloudwatch.ListMetricsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error) {
	return c.cloudwatch.ListMetrics(ctx, input, optFns...)
}
