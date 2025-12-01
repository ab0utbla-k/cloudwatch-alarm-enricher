# CloudWatch Alarm Enricher

AWS Lambda function that enriches CloudWatch alarm notifications with detailed metric analysis.

When a CloudWatch alarm triggers, this Lambda queries CloudWatch metrics to identify the exact resources violating the threshold and dispatches actionable notifications.

## Features

- **Metric Enrichment**: Identifies specific resources violating alarm thresholds
- **Multiple Dispatch Targets**: Send to SNS or EventBridge
- **Universal Support**: Works with any CloudWatch alarm
- **Flexible Deployment**: Deploy as zip package or container image

## How It Works

1. CloudWatch alarm state changes to `ALARM`
2. EventBridge triggers the Lambda function
3. Lambda enriches alarm with violating metrics
4. Lambda dispatches notification to configured target

## Configuration

### Environment Variables

| Variable            | Required         | Default | Description                             |
|---------------------|------------------|---------|-----------------------------------------|
| `ALARM_DESTINATION` | No               | `sns`   | Dispatch target: `sns` or `eventbridge` |
| `SNS_TOPIC_ARN`     | If `sns`         | -       | SNS topic ARN                           |
| `EVENT_BUS_ARN`     | If `eventbridge` | -       | EventBridge bus name or ARN             |

> **Note:** `AWS_REGION` is automatically provided by the Lambda runtime.

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
    },
    {
      "Effect": "Allow",
      "Action": [
        "xray:PutTraceSegments",
        "xray:PutTelemetryRecords"
      ],
      "Resource": "*"
    }
  ]
}
```

**Notes:**
- Add only SNS or EventBridge permissions based on your chosen dispatch target
- X-Ray permissions are required for distributed tracing
- Lambda tracing should be set to `PassThrough` mode to use OTEL instrumentation

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

### Zip Package

```bash
# Build and create deployment package
make lambda.build
make lambda.zip

# Deploy to AWS Lambda
make lambda.deploy

# Or specify custom function name
LAMBDA_FUNCTION_NAME=my-alarm-enricher make lambda.deploy

# Clean build artifacts
make lambda.clean
```

### Container Image

```bash
# Build container image
make docker.build

# Push to registry
make docker.push

# Customize image name/tag
IMAGE_REGISTRY=123456789.dkr.ecr.us-east-1.amazonaws.com \
IMAGE_REPO=cloudwatch-alarm-enricher \
IMAGE_TAG=v1.0.0 \
make docker.build docker.push
```

## Development

```bash
make test    # Run tests
make lint    # Run linter
make fmt     # Format code and fix lint issues
make tidy    # Tidy go modules
make help    # Show all available targets
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
