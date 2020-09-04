package observability

import (
	"fmt"
	"strings"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// NewPrometheusExporter builds and configures Prometheus exporter
func NewPrometheusExporter(serviceName string, serviceViews []*view.View) (*prometheus.Exporter, error) {
	prometheusNS := strings.ReplaceAll(serviceName, "-", "_")

	prometheusExporter, err := prometheus.NewExporter(prometheus.Options{Namespace: prometheusNS})
	if err != nil {
		return nil, err
	}

	view.RegisterExporter(prometheusExporter)
	view.SetReportingPeriod(time.Second)

	ocHTTPServerViews := []*view.View{
		ochttp.ServerRequestCountView,
		ochttp.ServerRequestBytesView,
		ochttp.ServerResponseBytesView,
		ochttp.ServerLatencyView,
		ochttp.ServerRequestCountByMethod,
		ochttp.ServerResponseCountByStatusCode,
		{
			Name:        "opencensus.io/http/server/latency_by_path",
			Description: "Latency distribution of HTTP requests by route",
			TagKeys:     []tag.Key{ochttp.KeyServerRoute},
			Measure:     ochttp.ServerLatency,
			Aggregation: ochttp.DefaultLatencyDistribution,
		},
	}

	vv := append(ocHTTPServerViews, serviceViews...)

	if err := view.Register(vv...); err != nil {
		return nil, fmt.Errorf("could not register prometheus server views: %w", err)
	}

	return prometheusExporter, nil
}
