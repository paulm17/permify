package meterexporters

import (
	"fmt"

	"go.opentelemetry.io/otel/sdk/metric"
)

// ExporterFactory - Create meter exporter according to given params
func ExporterFactory(name, endpoint string, insecure bool, urlpath string, headers map[string]string) (metric.Exporter, error) {
	switch name {
	case "otlp", "otlp-http":
		return NewOTLP(endpoint, insecure, urlpath, headers)
	case "otlp-grpc":
		return NewOTLPGrpc(endpoint, insecure, headers)
	default:
		return nil, fmt.Errorf("%s meter exporter is unsupported", name)
	}
}
