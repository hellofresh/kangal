package observability

import (
	"log"

	"go.opentelemetry.io/otel/exporters/prometheus"
)

func NewOtelPromExporter() *prometheus.Exporter {
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatal(err)
		return nil
	}
	//global.SetMeterProvider(NewMeterProvider())
	//
	//if err := runtime.Start(
	//	runtime.WithMinimumReadMemStatsInterval(time.Second),
	//); err != nil {
	//	return nil, fmt.Errorf("failed to start runtime instrumentation: %w", err)
	//}

	return exporter
}

// NewPrometheusExporter builds and configures Prometheus exporter
//func NewPrometheusExporter(serviceName string, serviceViews []*view.View) (*prometheus.Exporter, error) {
//	prometheusNS := strings.ReplaceAll(serviceName, "-", "_")
//
//	prometheusExporter, err := prometheus.NewExporter(prometheus.Options{Namespace: prometheusNS})
//	if err != nil {
//		return nil, err
//	}
//
//	view.RegisterExporter(prometheusExporter)
//	view.SetReportingPeriod(time.Second)
//
//	ocHTTPServerViews := []*view.View{
//		ochttp.ServerRequestCountView,
//		ochttp.ServerRequestBytesView,
//		ochttp.ServerResponseBytesView,
//		ochttp.ServerLatencyView,
//		ochttp.ServerRequestCountByMethod,
//		ochttp.ServerResponseCountByStatusCode,
//		{
//			Name:        "opencensus.io/http/server/latency_by_path",
//			Description: "Latency distribution of HTTP requests by route",
//			TagKeys:     []tag.Key{ochttp.KeyServerRoute},
//			Measure:     ochttp.ServerLatency,
//			Aggregation: ochttp.DefaultLatencyDistribution,
//		},
//	}
//
//	vv := append(ocHTTPServerViews, serviceViews...)
//
//	if err := view.Register(vv...); err != nil {
//		return nil, fmt.Errorf("could not register prometheus server views: %w", err)
//	}
//
//	return prometheusExporter, nil
//}
