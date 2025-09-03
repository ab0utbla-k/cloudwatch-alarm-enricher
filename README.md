# CloudWatch Alarm Enricher

A Go-based AWS Lambda function that enhances CloudWatch alarm notifications with detailed metric analysis and context. This solution processes CloudWatch alarm state changes and enriches notifications with granular information about which specific resources are violating alarm thresholds.

## Overview

When CloudWatch alarms trigger, they often provide limited context about which specific resources are causing the threshold violations. This Lambda function intercepts alarm state change events and enriches the notifications with detailed metric breakdowns, helping operators quickly identify root causes.

## Features

- **Universal Alarm Support**: Works with any CloudWatch alarm across AWS services (EKS, EC2, RDS, etc.)
- **Metric Enrichment**: Identifies specific resources violating thresholds by querying CloudWatch metrics
- **Flexible Notifications**: Sends enriched alerts via Amazon SNS with detailed violation context
- **Event-Driven Architecture**: Automatically triggered by CloudWatch alarm state changes via EventBridge
- **Scalable Structure**: Modular Go architecture designed for easy extension and maintenance

## Architecture

```
CloudWatch Alarm → EventBridge Rule → Lambda Function → SNS Topic → Email/SMS
                                    ↓
                               CloudWatch Metrics API
                              (enrichment queries)
```

