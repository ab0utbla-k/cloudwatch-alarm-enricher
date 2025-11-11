# CloudWatch Alarm Enricher

AWS Lambda function that enriches CloudWatch alarm notifications with detailed metric analysis.

When a CloudWatch alarm triggers, this Lambda queries CloudWatch metrics to identify the exact resources violating the threshold and dispatches actionable notifications.

## Features

- **Metric Enrichment**: Identifies specific resources violating alarm thresholds
- **Multiple Dispatch Targets**: Send to SNS or EventBridge
- **Universal Support**: Works with any CloudWatch alarm
- **Container-based**: Deployed as Lambda container image

## How It Works

1. CloudWatch alarm state changes to `ALARM`
2. EventBridge triggers the Lambda function
3. Lambda enriches alarm with violating metrics
4. Lambda dispatches notification to configured target

## Configuration

### Environment Variables

**Required:**
- `AWS_REGION`: AWS region (e.g., `us-east-1`)
- `ALARM_DESTINATION`: Dispatch target - `sns` or `eventbridge` (default: `sns`)

**Target-specific:**
- `SNS_TOPIC_ARN`: SNS topic ARN (required if `ALARM_DESTINATION=sns`)
- `EVENT_BUS_ARN`: EventBridge bus name or ARN (required if `ALARM_DESTINATION=eventbridge`)

### IAM Permissions

Lambda execution role needs:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudwatch:DescribeAlarms",
        "cloudwatch:GetMetricData",
        "cloudwatch:ListMetrics"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": ["sns:Publish"],
      "Resource": "arn:aws:sns:REGION:ACCOUNT:TOPIC_NAME"
    },
    {
      "Effect": "Allow",
      "Action": ["events:PutEvents"],
      "Resource": "arn:aws:events:REGION:ACCOUNT:event-bus/BUS_NAME"
    }
  ]
}
```

Add only the permissions needed for your chosen dispatch target.

### EventBridge Rule

Trigger Lambda on CloudWatch alarm state changes:

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

# Deploy to AWS Lambda
make deploy

# Or specify custom function name
LAMBDA_FUNCTION_NAME=my-alarm-enricher make deploy

# Clean build artifacts
make clean
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Using Makefile
make test
```

### Code Quality

```bash
# Lint code
make lint

# Format and fix code
make fmt

# Tidy modules
make tidy
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

## License

Apache License 2.0 - see [LICENSE](LICENSE)
