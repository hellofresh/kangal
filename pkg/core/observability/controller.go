package observability

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"time"

	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
)

var (
	// reconcileDistribution defines the bucket boundaries for the histogram of reconcile latency metric
	// Bucket boundaries are 10ms, 100ms, 1s, 10s, 30s and 60s.
	reconcileDistribution = []float64{10, 100, 1000, 10000, 30000, 60000}
)

type MetricsReporter struct {
	countRunningLoadtests syncint64.UpDownCounter
	workQueueDepthStat    syncint64.UpDownCounter
	reconcileCountStat    syncint64.UpDownCounter
	reconcileLatencyStat  syncint64.Histogram
}

func NewMeterProvider() *metric.MeterProvider {
	return metric.NewMeterProvider(
		metric.WithReader(NewOtelPromExporter()),
		metric.WithView(metric.NewView(
			metric.Instrument{Name: "kangal_reconcile_latency"},
			metric.Stream{Aggregation: aggregation.ExplicitBucketHistogram{
				Boundaries: reconcileDistribution,
			}},
		)),
		//metric.WithView(metric.NewView(
		//	metric.Instrument{Name: "running_loadtests_count"},
		//	metric.Stream{Aggregation: aggregation.LastValue{}},
		//)),
	)
}

func NewMetricReporter() (*MetricsReporter, error) {
	meter := NewMeterProvider().Meter("https://github.com/hellofresh/kangal/pkg/core/observability")

	countRunningLoadtests, err := meter.SyncInt64().UpDownCounter(
		"kangal_running_loadtests_count",
		instrument.WithDescription("The number of currently running loadtests"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		fmt.Errorf("could not register countRunningLoadtests metric: %w", err)
		return nil, err
	}

	//if err := meter.RegisterCallback(
	//	[]instrument.Asynchronous{
	//		countRunningLoadtests,
	//	},
	//	func(ctx context.Context) {
	//		countRunningLoadtests.Observe(context.Background(), 1, attribute.String("loadtest", "running"))
	//	},
	//); err != nil {
	//	panic(err)
	//}

	workQueueDepthStat, err := meter.SyncInt64().UpDownCounter(
		"kangal_work_queue_depth",
		instrument.WithDescription("Depth of the work queue"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		fmt.Errorf("could not register workQueueDepthStat metric: %w", err)
		return nil, err
	}

	reconcileCountStat, err := meter.SyncInt64().UpDownCounter(
		"kangal_reconcile_count",
		instrument.WithDescription("Number of reconcile operations"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		fmt.Errorf("could not register reconcileCountStat metric: %w", err)
		return nil, err
	}

	reconcileLatencyStat, err := meter.SyncInt64().Histogram(
		"kangal_reconcile_latency",
		instrument.WithDescription("Latency of reconcile operations"),
		instrument.WithUnit(unit.Milliseconds),
	)
	if err != nil {
		fmt.Errorf("could not register reconcileLatencyStat metric: %w", err)
		return nil, err
	}

	return &MetricsReporter{
		countRunningLoadtests: countRunningLoadtests,
		workQueueDepthStat:    workQueueDepthStat,
		reconcileCountStat:    reconcileCountStat,
		reconcileLatencyStat:  reconcileLatencyStat,
	}, nil
}

func (r *MetricsReporter) AddRunningLTCounter(ltCount int64) {
	r.countRunningLoadtests.Add(context.Background(), ltCount, attribute.String("loadtest", "running"))
}

func (r *MetricsReporter) ReportReconcile(duration time.Duration, key, success string) {
	r.reconcileCountStat.Add(context.Background(), 1, attribute.String("key", key), attribute.String("success", success))
	r.reconcileLatencyStat.Record(context.Background(), int64(duration/time.Millisecond), attribute.String("key", key), attribute.String("success", success))
}

func (r *MetricsReporter) ReportWorkQueueDepth(v int64) {
	r.workQueueDepthStat.Add(context.Background(), v)
}
