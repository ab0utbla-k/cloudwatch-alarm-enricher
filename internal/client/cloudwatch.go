package client

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

// CloudWatchClient wraps the AWS CloudWatch client and implements CloudWatchAPI interface.
type CloudWatchClient struct {
	client *cloudwatch.Client
}

func NewCloudWatch(client *cloudwatch.Client) *CloudWatchClient {
	return &CloudWatchClient{
		client: client,
	}
}

// DescribeAlarms retrieves information about CloudWatch alarms.
func (c *CloudWatchClient) DescribeAlarms(ctx context.Context, input *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	return c.client.DescribeAlarms(ctx, input, optFns...)
}

// GetMetricData retrieves metric data for CloudWatch metrics.
func (c *CloudWatchClient) GetMetricData(ctx context.Context, input *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	return c.client.GetMetricData(ctx, input, optFns...)
}

// ListMetrics retrieves a list of CloudWatch metrics.
func (c *CloudWatchClient) ListMetrics(ctx context.Context, input *cloudwatch.ListMetricsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error) {
	return c.client.ListMetrics(ctx, input, optFns...)
}
