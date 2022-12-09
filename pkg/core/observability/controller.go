package observability

import (
	"context"
	"fmt"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"time"

	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
)

//var (
//	workQueueDepthStat   = stats.Int64("work_queue_depth", "Depth of the work queue", stats.UnitDimensionless)
//	reconcileCountStat   = stats.Int64("reconcile_count", "Number of reconcile operations", stats.UnitDimensionless)
//	reconcileLatencyStat = stats.Int64("reconcile_latency", "Latency of reconcile operations", stats.UnitMilliseconds)
//
//	// MRunningLoadtestCountStat counts the number of running loadtests
//	MRunningLoadtestCountStat = stats.Int64("running_loadtests_count", "Number of running loadtests", stats.UnitDimensionless)
//
//
//)

var (
	// reconcileDistribution defines the bucket boundaries for the histogram of reconcile latency metric
	// Bucket boundaries are 10ms, 100ms, 1s, 10s, 30s and 60s.
	reconcileDistribution = []float64{10, 100, 1000, 10000, 30000, 60000}

	// Create the tag keys that will be used to add tags to our measurements.
	reconcilerTagKey = mustNewTagKey("reconciler")
	keyTagKey        = mustNewTagKey("key")
	successTagKey    = mustNewTagKey("success")
)

type MetricsReporter struct {
	mCountRunningLt      syncint64.UpDownCounter
	workQueueDepthStat   syncint64.UpDownCounter
	reconcileCountStat   syncint64.UpDownCounter
	reconcileLatencyStat syncint64.Histogram
	reconciler           string
	globalCtx            context.Context
}

func NewMeter() *metric.MeterProvider {
	//return global.MeterProvider()
	return metric.NewMeterProvider(
		metric.WithReader(NewOtelPromExporter()),
		metric.WithView(metric.NewView(
			metric.Instrument{Name: "reconcile_latency"},
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
	meter := NewMeter().Meter("https://github.com/hellofresh/kangal/pkg/core/observability")
	//histogram := aggregation.ExplicitBucketHistogram{
	//	Boundaries: reconcileDistribution,
	//	NoMinMax:   false,
	//}

	mCountRunninLt, err := meter.SyncInt64().UpDownCounter(
		"running_loadtests_count",
		instrument.WithDescription("The number of currently running loadtests"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		fmt.Errorf("could not register mCounTRunninLt metric: %w", err)
		return nil, err
	}

	workQueueDepthStat, err := meter.SyncInt64().UpDownCounter(
		"work_queue_depth",
		instrument.WithDescription("Depth of the work queue"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		fmt.Errorf("could not register workQueueDepthStat metric: %w", err)
		return nil, err
	}

	reconcileCountStat, err := meter.SyncInt64().UpDownCounter(
		"reconcile_count",
		instrument.WithDescription("Number of reconcile operations"),
		instrument.WithUnit(unit.Dimensionless),
	)
	if err != nil {
		fmt.Errorf("could not register reconcileCountStat metric: %w", err)
		return nil, err
	}

	reconcileLatencyStat, err := meter.SyncInt64().Histogram(
		"reconcile_latency",
		instrument.WithDescription("Latency of reconcile operations"),
		instrument.WithUnit(unit.Milliseconds),
	)
	if err != nil {
		fmt.Errorf("could not register reconcileLatencyStat metric: %w", err)
		return nil, err
	}

	return &MetricsReporter{
		mCountRunningLt:      mCountRunninLt,
		workQueueDepthStat:   workQueueDepthStat,
		reconcileCountStat:   reconcileCountStat,
		reconcileLatencyStat: reconcileLatencyStat,
		reconciler:           "kangal",
	}, nil
}

func (r *MetricsReporter) AddRunningLTCounter(ltCount int64) {
	r.mCountRunningLt.Add(context.Background(), ltCount, attribute.String("loadtest", "running"))
}

func (r *MetricsReporter) ReportReconcile(duration time.Duration, key, success string) error {
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(reconcilerTagKey, r.reconciler),
		tag.Insert(keyTagKey, key),
		tag.Insert(successTagKey, success))
	if err != nil {
		return err
	}

	r.reconcileCountStat.Add(ctx, 1, attribute.String("key", key), attribute.String("reconciler", r.reconciler), attribute.String("success", success))
	r.reconcileLatencyStat.Record(ctx, int64(duration/time.Millisecond), attribute.String("key", key), attribute.String("Key", key), attribute.String("reconciler", r.reconciler), attribute.String("success", success))

	return nil
}

// ControllerViews are the views needed to be registered for getting metrics
// from the kangal controller
//var ControllerViews = []*view.View{
//	{
//		Description: "Depth of the work queue",
//		Measure:     workQueueDepthStat,
//		Aggregation: view.LastValue(),
//		TagKeys:     []tag.Key{reconcilerTagKey},
//	},
//	{
//		Description: "Number of reconcile operations",
//		Measure:     reconcileCountStat,
//		Aggregation: view.Count(),
//		TagKeys:     []tag.Key{reconcilerTagKey, keyTagKey, successTagKey},
//	},
//	{
//		Description: "Latency of reconcile operations",
//		Measure:     reconcileLatencyStat,
//		Aggregation: reconcileDistribution,
//		TagKeys:     []tag.Key{reconcilerTagKey, keyTagKey, successTagKey},
//	},
//	{
//		Description: MRunningLoadtestCountStat.Description(),
//		Measure:     MRunningLoadtestCountStat,
//		Aggregation: view.LastValue(),
//	},
//}

// StatsReporter defines the interface for sending metrics
//type StatsReporter interface {
//	// ReportQueueDepth reports the queue depth metric
//	ReportQueueDepth(v int64) error
//
//	// ReportReconcile reports the count and latency metrics for a reconcile operation
//	ReportReconcile(duration time.Duration, key, success string) error
//}

// Reporter holds cached metric objects to report metrics
//type reporter struct {
//	reconciler string
//	globalCtx  context.Context
//}

// NewStatsReporter creates a reporter that collects and reports metrics
//func NewStatsReporter(reconciler string) (StatsReporter, error) {
//	// Reconciler tag is static. Create a context containing that and cache it.
//	ctx, err := tag.New(
//		context.Background(),
//		tag.Insert(reconcilerTagKey, reconciler))
//	if err != nil {
//		return nil, err
//	}
//
//	return &reporter{reconciler: reconciler, globalCtx: ctx}, nil
//}

// ReportQueueDepth reports the queue depth metric
//func (r *reporter) ReportQueueDepth(v int64) error {
//	if r.globalCtx == nil {
//		return errors.New("reporter is not initialized correctly")
//	}
//	stats.Record(r.globalCtx, workQueueDepthStat.M(v))
//	return nil
//}

// ReportReconcile reports the count and latency metrics for a reconcile operation
//func (r *reporter) ReportReconcile(duration time.Duration, key, success string) error {
//	ctx, err := tag.New(
//		context.Background(),
//		tag.Insert(reconcilerTagKey, r.reconciler),
//		tag.Insert(keyTagKey, key),
//		tag.Insert(successTagKey, success))
//	if err != nil {
//		return err
//	}
//
//	stats.Record(ctx, reconcileCountStat.M(1))
//	stats.Record(ctx, reconcileLatencyStat.M(int64(duration/time.Millisecond)))
//	return nil
//}

func mustNewTagKey(s string) tag.Key {
	tagKey, err := tag.NewKey(s)
	if err != nil {
		panic(err)
	}
	return tagKey
}
