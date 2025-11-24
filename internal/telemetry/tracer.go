package telemetry

import (
	"context"
	"fmt"
	"os"

	"github.com/aws-observability/aws-otel-go/exporters/xrayudp"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func NewTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	res, err := buildResource(ctx)
	if err != nil {
		return nil, err
	}

	exp, err := xrayudp.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot create xray udp exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	return tp, nil
}

// buildResource creates a merged OTEL resource with Lambda detection and custom attributes.
func buildResource(ctx context.Context) (*resource.Resource, error) {
	detector := lambdadetector.NewResourceDetector()
	lambdaResource, err := detector.Detect(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot detect lambda resource: %w", err)
	}

	attributes := []attribute.KeyValue{
		{
			Key:   semconv.ServiceNameKey,
			Value: attribute.StringValue(os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
		},
	}
	customResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)

	mergedResource, err := resource.Merge(lambdaResource, customResource)
	if err != nil {
		return nil, fmt.Errorf("cannot merge otel resources: %w", err)
	}

	return mergedResource, nil
}
