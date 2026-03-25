package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds tracing configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string // e.g., "localhost:4317"
	Enabled        bool
}

// InitTracer initializes OpenTelemetry tracing
func InitTracer(cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		// Return no-op shutdown function
		return func(context.Context) error { return nil }, nil
	}

	// Create OTLP exporter
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		cfg.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to OTLP: %w", err)
	}

	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(otlptracegrpc.WithGRPCConn(conn)))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // For MVP, sample everything
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Set global propagator (for cross-service trace context)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return shutdown function
	return tp.Shutdown, nil
}

// Tracer returns a tracer for the given service
func Tracer(serviceName string) trace.Tracer {
	return otel.Tracer(serviceName)
}

// StartSpan starts a new span with common attributes
func StartSpan(ctx context.Context, tracer trace.Tracer, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, spanName)
	span.SetAttributes(attrs...)
	return ctx, span
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// Common attribute keys
const (
	AttrTenantID   = "tenant.id"
	AttrRequestID  = "request.id"
	AttrEventType  = "event.type"
	AttrRuleID     = "rule.id"
	AttrAlertID    = "alert.id"
	AttrEntityID   = "entity.id"
	AttrNodeID     = "graph.node.id"
	AttrEdgeCount  = "graph.edge.count"
)

// Helper functions for common attributes
func TenantID(id string) attribute.KeyValue {
	return attribute.String(AttrTenantID, id)
}

func RequestID(id string) attribute.KeyValue {
	return attribute.String(AttrRequestID, id)
}

func EventType(typ string) attribute.KeyValue {
	return attribute.String(AttrEventType, typ)
}

func RuleID(id string) attribute.KeyValue {
	return attribute.String(AttrRuleID, id)
}

func AlertID(id string) attribute.KeyValue {
	return attribute.String(AttrAlertID, id)
}

func EntityID(id string) attribute.KeyValue {
	return attribute.String(AttrEntityID, id)
}

func NodeID(id string) attribute.KeyValue {
	return attribute.String(AttrNodeID, id)
}

func EdgeCount(count int) attribute.KeyValue {
	return attribute.Int(AttrEdgeCount, count)
}
