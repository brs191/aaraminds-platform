package aapruntime

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelInstrumentationName = "github.com/aaraminds/aaraminds-platform/platform/internal/runtime"
	defaultServiceName      = "aaraminds-aap-runtime"
)

// OTelConfig configures the optional OpenTelemetry exporter used by the local
// proof harness. The runtime still treats audit events as the control plane;
// spans are an observability projection of the existing trace-shaped records.
type OTelConfig struct {
	Enabled        bool
	Exporter       string
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Insecure       bool
}

type otelRunState struct {
	ctx   context.Context
	span  trace.Span
	ended bool
}

// OTelConfigFromEnv returns a quiet default configuration. OTel stays disabled
// unless AAP_OTEL_ENABLED is truthy or the CLI sets Enabled directly.
func OTelConfigFromEnv(serviceName, serviceVersion string) OTelConfig {
	if serviceName == "" {
		serviceName = defaultServiceName
	}
	endpoint := firstEnv("AAP_OTEL_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT")
	insecureDefault := true
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(endpoint)), "https://") {
		insecureDefault = false
	}
	cfg := OTelConfig{
		Enabled:        envBool("AAP_OTEL_ENABLED", false),
		Exporter:       firstEnv("AAP_OTEL_EXPORTER", "OTEL_TRACES_EXPORTER"),
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    firstEnv("AAP_ENVIRONMENT", "OTEL_DEPLOYMENT_ENVIRONMENT"),
		Endpoint:       endpoint,
		Insecure:       envBool("AAP_OTEL_INSECURE", insecureDefault),
	}
	if cfg.Environment == "" {
		cfg.Environment = resourceAttributeFromEnv("deployment.environment.name")
	}
	if cfg.Exporter == "" {
		cfg.Exporter = "stdout"
	}
	return cfg
}

// ConfigureOpenTelemetry installs an SDK tracer provider for the process and
// returns a shutdown function. Supported exporters are "stdout", "otlp", and
// "none". The OTLP exporter also honors standard OTEL_* environment variables.
func ConfigureOpenTelemetry(ctx context.Context, cfg OTelConfig) (func(context.Context) error, error) {
	if !cfg.Enabled || strings.EqualFold(cfg.Exporter, "none") {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = defaultServiceName
	}

	exp, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	resourceAttrs := []attribute.KeyValue{attribute.String("service.name", cfg.ServiceName)}
	if cfg.ServiceVersion != "" {
		resourceAttrs = append(resourceAttrs, attribute.String("service.version", cfg.ServiceVersion))
	}
	if cfg.Environment != "" {
		resourceAttrs = append(resourceAttrs, attribute.String("deployment.environment.name", cfg.Environment))
	}
	res, err := resource.New(ctx, resource.WithAttributes(resourceAttrs...))
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}

func newTraceExporter(ctx context.Context, cfg OTelConfig) (sdktrace.SpanExporter, error) {
	switch strings.ToLower(cfg.Exporter) {
	case "stdout", "console":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otlp", "otlp-grpc":
		opts := []otlptracegrpc.Option{}
		if cfg.Endpoint != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(stripEndpointScheme(cfg.Endpoint)))
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported otel exporter %q", cfg.Exporter)
	}
}

// EndRun closes the run-scoped OTel root span. It is idempotent; audit and
// local trace records remain the source of truth for run lifecycle state.
func (e *Engine) EndRun(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_ = ctx
	e.endOTelRun(time.Now().UTC())
	return nil
}

func (e *Engine) startOTelRun(start time.Time) {
	if !e.manifest.Telemetry.OTELEnabled {
		return
	}
	attrs := e.baseOTelAttributes("invoke_agent")
	tracer := otel.Tracer(otelInstrumentationName)
	ctx, span := tracer.Start(context.Background(), "invoke_agent "+e.manifest.AgentID,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithTimestamp(start),
		trace.WithAttributes(attrs...),
	)
	e.otelRun = otelRunState{ctx: ctx, span: span}
}

func (e *Engine) endOTelRun(end time.Time) {
	if e.otelRun.span == nil || e.otelRun.ended {
		return
	}
	e.otelRun.span.End(trace.WithTimestamp(end))
	e.otelRun.ended = true
}

func (e *Engine) emitOTelSpan(name, kind string, attrs map[string]any, start, end time.Time) {
	if !e.manifest.Telemetry.OTELEnabled {
		return
	}
	if name == "agent.run" {
		return
	}
	spanKind := trace.SpanKindInternal
	if kind == "tool" {
		spanKind = trace.SpanKindClient
	}
	otelAttrs := e.otelAttributes(name, attrs)
	tracer := otel.Tracer(otelInstrumentationName)
	parent := e.otelRun.ctx
	if parent == nil {
		parent = context.Background()
	}
	_, span := tracer.Start(parent, name,
		trace.WithSpanKind(spanKind),
		trace.WithTimestamp(start),
		trace.WithAttributes(otelAttrs...),
	)
	span.End(trace.WithTimestamp(end))
}

func (e *Engine) otelAttributes(spanName string, attrs map[string]any) []attribute.KeyValue {
	out := e.baseOTelAttributes(genAIOperationName(spanName))
	if toolName, ok := attrs["tool_name"]; ok {
		out = append(out,
			attribute.String("gen_ai.tool.name", fmt.Sprint(toolName)),
			attribute.String("gen_ai.tool.type", "function"),
		)
	}

	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out = append(out, otelAttribute("aap."+key, attrs[key]))
	}
	return out
}

func (e *Engine) baseOTelAttributes(operationName string) []attribute.KeyValue {
	out := []attribute.KeyValue{
		attribute.String("aap.run_id", e.run.RunID),
		attribute.String("aap.engagement_id", e.run.EngagementID),
		attribute.String("aap.agent_id", e.run.AgentID),
		attribute.String("aap.manifest_version", e.run.ManifestVersion),
		attribute.String("aap.tenant_namespace", e.run.TenantNamespace),
		attribute.String("aap.workflow.name", "aap.local_runtime_proof"),
		attribute.String("gen_ai.agent.id", e.run.AgentID),
		attribute.String("gen_ai.agent.name", e.manifest.AgentID),
	}
	if operationName != "" {
		out = append(out, attribute.String("gen_ai.operation.name", operationName))
	}
	return out
}

func otelAttribute(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case Boundary:
		return attribute.String(key, string(v))
	case RuntimeMode:
		return attribute.String(key, string(v))
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case time.Duration:
		return attribute.Int64(key+".ms", v.Milliseconds())
	case []string:
		return attribute.StringSlice(key, v)
	default:
		return attribute.String(key, fmt.Sprint(v))
	}
}

func genAIOperationName(spanName string) string {
	switch {
	case spanName == "agent.run":
		return "invoke_agent"
	case spanName == "tool.invoked":
		return "execute_tool"
	default:
		return ""
	}
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func resourceAttributeFromEnv(name string) string {
	for _, part := range strings.Split(os.Getenv("OTEL_RESOURCE_ATTRIBUTES"), ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && key == name {
			return value
		}
	}
	return ""
}

func stripEndpointScheme(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	for _, prefix := range []string{"http://", "https://"} {
		endpoint = strings.TrimPrefix(endpoint, prefix)
	}
	return strings.TrimRight(endpoint, "/")
}
