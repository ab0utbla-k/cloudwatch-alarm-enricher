# CloudWatch Alarm Enricher

AWS Lambda function that enriches CloudWatch alarm notifications with detailed metric analysis and resource-specific context.

## Features

- **Metric Enrichment**: Identifies specific resources violating alarm thresholds
- **Universal Support**: Works with all CloudWatch alarms (EC2, EKS, RDS, etc.)
- **Event-Driven**: Triggered by CloudWatch alarm state changes via EventBridge
- **SNS Integration**: Sends enriched notifications via Amazon SNS

## Architecture

```
CloudWatch Alarm → EventBridge → Lambda → SNS → Notifications
                                   ↓
                             CloudWatch API
                           (metric queries)
```

## Configuration

Required environment variables:
- `AWS_REGION`: AWS region
- `SNS_TOPIC_ARN`: SNS topic for notifications

## Deployment

```bash
# Build
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bootstrap ./cmd/lambda

# Deploy using your preferred method (SAM, CDK, Terraform, etc.)
```

## Development

```bash
# Lint code
make lint

# Format code
make fmt
```
