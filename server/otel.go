package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.com/logtrace/logtrace/config"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("logtrace.server")

func getTracer(ctx context.Context, r *http.Request,
	operationName string,
) (context.Context, trace.Span, string) {
	rid := retrieveRequestID(r)
	ctx, span := tracer.Start(ctx, operationName)

	span.SetAttributes(attribute.String("request_id", rid))
	return ctx, span, rid
}

func initResources() (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", "logtrace"),
			attribute.String("library.language", "go"),
		),
	)
}

func InitOTELCapabilities(cfg config.Config, logger *zap.Logger) (func(), http.Handler) {
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{}),
	)

	resources, err := initResources()
	if err != nil {
		logger.Fatal("could not setup OTEL tracing resources",
			zap.Error(err))
	}

	var (
		tracesSuffixEndpoint  = "/v1/traces"
		metricsSuffixEndpoint = "/v1/metrics"
	)

	headers := map[string]string{}
	pairs := strings.Split(cfg.Otel.Headers, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	// By default, Otel sends traces and metrics, logs to v1/* paths
	// but some providers like Grafana have their OTEL collector on a subpath
	// so /otlp/v1/*
	// The gRPC exporter expects host:port with no scheme, so strip http(s):// if present
	// and handle any subpath prefix.
	if parsedURL, parseErr := url.Parse(cfg.Otel.Endpoint); parseErr == nil && parsedURL.Host != "" {
		cfg.Otel.Endpoint = parsedURL.Host
		if subpath := strings.TrimSuffix(parsedURL.Path, "/"); subpath != "" {
			tracesSuffixEndpoint = subpath + tracesSuffixEndpoint
			metricsSuffixEndpoint = subpath + metricsSuffixEndpoint
		}
	} else {
		// Fallback: bare host:port/subpath format (no scheme)
		splittedEndpoint := strings.SplitN(cfg.Otel.Endpoint, "/", 2)
		if len(splittedEndpoint) == 2 {
			cfg.Otel.Endpoint = splittedEndpoint[0]
			tracesSuffixEndpoint = "/" + splittedEndpoint[1] + tracesSuffixEndpoint
			metricsSuffixEndpoint = "/" + splittedEndpoint[1] + metricsSuffixEndpoint
		}
	}

	// If no endpoint is configured, skip gRPC exporters and only expose Prometheus metrics.
	if cfg.Otel.Endpoint == "" {
		logger.Warn("OTEL endpoint not configured; traces and OTLP metrics export disabled")

		promExporter, err := otelprometheus.New()
		if err != nil {
			logger.Fatal("could not setup prometheus metrics exporter",
				zap.Error(err))
		}

		otel.SetMeterProvider(
			metric.NewMeterProvider(
				metric.WithResource(resources),
				metric.WithReader(promExporter),
			))

		regiterMetrics(logger)

		return func() {}, promhttp.Handler()
	}

	traceOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Otel.Endpoint),
		otlptracegrpc.WithHeaders(headers),
	}

	if !cfg.Otel.UseTLS {
		traceOptions = append(traceOptions, otlptracegrpc.WithInsecure())
	}

	traceExporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(traceOptions...))
	if err != nil {
		logger.Fatal("could not setup OTEL tracing resources",
			zap.Error(err))
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(traceExporter,
				sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
				sdktrace.WithBatchTimeout(5*time.Second),
			),
			sdktrace.WithResource(resources),
		),
	)

	metricsOptions := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.Otel.Endpoint),
		otlpmetricgrpc.WithHeaders(headers),
	}

	if !cfg.Otel.UseTLS {
		metricsOptions = append(metricsOptions, otlpmetricgrpc.WithInsecure())
	}

	metricExporter, err := otlpmetricgrpc.New(
		context.Background(), metricsOptions...)
	if err != nil {
		logger.Fatal("could not setup metrics exporter",
			zap.Error(err))
	}

	promExporter, err := otelprometheus.New()
	if err != nil {
		logger.Fatal("could not setup prometheus metrics exporter",
			zap.Error(err))
	}

	otel.SetMeterProvider(
		metric.NewMeterProvider(
			metric.WithResource(resources),
			metric.WithReader(
				metric.NewPeriodicReader(metricExporter)),
			metric.WithReader(promExporter),
		))

	regiterMetrics(logger)

	return func() {
		_ = traceExporter.Shutdown(context.Background())
		_ = metricExporter.Shutdown(context.Background())
	}, promhttp.Handler()
}

func regiterMetrics(logger *zap.Logger) {
	err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		logger.Fatal("could not gather runtime metrics",
			zap.Error(err))
	}
}
