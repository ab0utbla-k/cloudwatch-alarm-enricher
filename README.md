# CloudWatch Alarm Enricher

AWS Lambda function that enriches CloudWatch alarm notifications with detailed metric analysis and resource-specific context.

When a CloudWatch alarm triggers, this Lambda function queries CloudWatch metrics to identify the exact resources (EC2 instances, EKS pods, RDS databases, etc.) that are violating the alarm threshold, providing actionable insights instead of generic alerts.

## Features

- **Metric Enrichment**: Identifies specific resources violating alarm thresholds
- **Universal Support**: Works with all CloudWatch alarms (EC2, EKS, RDS, etc.)
- **Event-Driven**: Triggered by CloudWatch alarm state changes via EventBridge
- **SNS Integration**: Sends enriched notifications via Amazon SNS

## How It Works

1. CloudWatch alarm state changes to `ALARM`
2. EventBridge captures the event and triggers the Lambda
3. Lambda fetches alarm details from CloudWatch
4. Lambda queries metric data to identify which specific resources are violating the threshold
5. Lambda sends an enriched notification via SNS with detailed breakdown

## Architecture

```
CloudWatch Alarm → EventBridge → Lambda → SNS → Notifications
                                   ↓
                             CloudWatch API
                           (metric queries)
```

## Configuration

### Environment Variables

Required environment variables for Lambda function:
- `AWS_REGION`: AWS region (e.g., `us-east-1`)
- `SNS_TOPIC_ARN`: SNS topic ARN for notifications (e.g., `arn:aws:sns:us-east-1:123456789012:alarm-notifications`)

### IAM Permissions

The Lambda execution role requires the following permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudwatch:ListMetrics",
        "cloudwatch:GetMetricData",
        "cloudwatch:DescribeAlarms"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "sns:Publish"
      ],
      "Resource": "arn:aws:sns:REGION:ACCOUNT:TOPIC_NAME"
    }
  ]
}
```

### EventBridge Rule

Create an EventBridge rule to trigger the Lambda on CloudWatch alarm state changes:

**Event Pattern:**
```json
{
  "source": ["aws.cloudwatch"],
  "detail-type": ["CloudWatch Alarm State Change"],
  "detail": {
    "state": {
      "value": ["ALARM"]
    }
  }
}
```

## Deployment

### Using Makefile

```bash
# Build Lambda binary
make build

# Create deployment package
make zip

# Deploy to AWS Lambda (requires LAMBDA_FUNCTION_NAME env var or default name)
make deploy

# Or specify custom function name
LAMBDA_FUNCTION_NAME=my-alarm-enricher make deploy

# Clean build artifacts
make clean
```

### Using IaC Tools

Deploy using your preferred Infrastructure as Code tool:
- AWS CDK
- Terraform
- CloudFormation

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests verbosely
go test -v ./...
```

### Code Quality

```bash
# Lint code
make lint

# Format and fix code
make fmt

# Both will install golangci-lint locally if not present
```

## Example Output

**Before (Standard CloudWatch Alarm):**
```
You are receiving this email because your Amazon CloudWatch Alarm "eks-node-high-filesystem" in the EU (Stockholm) region has entered the ALARM state, because "Threshold Crossed: 5 datapoints were greater than or equal to the threshold (80.0). The most recent datapoints which crossed the threshold: [84.93923823723915 (03/09/25 06:24:00), 84.9368272080435 (03/09/25 06:23:00), 84.93257920422259 (03/09/25 06:22:00), 84.9314310950818 (03/09/25 06:21:00), 84.92614979303418 (03/09/25 06:20:00)]." at "Wednesday 03 September, 2025 06:25:33 UTC".
```

**After (Enriched Notification):**
```
🚨 CloudWatch Alarm: pod-status-pending
State: ALARM
Reason: Threshold Crossed: 3 out of the last 3 datapoints [1.0 (03/10/25 16:12:00), 1.0 (03/10/25 16:11:00), 1.0 (03/10/25 16:10:00)] were greater than or equal to the threshold (1.0) (minimum 3 datapoints for OK -> ALARM transition).

Metrics currently violating (>= 1.0) threshold:
1. ClusterName=eks-test, FullPodName=test-failed-pod, Namespace=default, PodName=test-failed-pod, Value: 1.00
2. ClusterName=eks-test, FullPodName=test-failed-pod, Namespace=kube-system, PodName=test-failed-pod, Value: 1.00

Timestamp: 2025-10-03T16:13:52Z
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
