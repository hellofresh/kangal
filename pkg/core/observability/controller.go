package observability

import (
	"context"
	"errors"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	workQueueDepthStat   = stats.Int64("work_queue_depth", "Depth of the work queue", stats.UnitDimensionless)
	reconcileCountStat   = stats.Int64("reconcile_count", "Number of reconcile operations", stats.UnitDimensionless)
	reconcileLatencyStat = stats.Int64("reconcile_latency", "Latency of reconcile operations", stats.UnitMilliseconds)

	// MRunningLoadtestCountStat counts the number of running loadtests
	MRunningLoadtestCountStat = stats.Int64("running_loadtests_count", "Number of running loadtests", stats.UnitDimensionless)

	// reconcileDistribution defines the bucket boundaries for the histogram of reconcile latency metric.
	// Bucket boundaries are 10ms, 100ms, 1s, 10s, 30s and 60s.
	reconcileDistribution = view.Distribution(10, 100, 1000, 10000, 30000, 60000)

	// Create the tag keys that will be used to add tags to our measurements.
	reconcilerTagKey = mustNewTagKey("reconciler")
	keyTagKey        = mustNewTagKey("key")
	successTagKey    = mustNewTagKey("success")
)

// ControllerViews are the views needed to be registered for getting metrics
// from the kangal controller
var ControllerViews = []*view.View{
	{
		Description: "Depth of the work queue",
		Measure:     workQueueDepthStat,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{reconcilerTagKey},
	},
	{
		Description: "Number of reconcile operations",
		Measure:     reconcileCountStat,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{reconcilerTagKey, keyTagKey, successTagKey},
	},
	{
		Description: "Latency of reconcile operations",
		Measure:     reconcileLatencyStat,
		Aggregation: reconcileDistribution,
		TagKeys:     []tag.Key{reconcilerTagKey, keyTagKey, successTagKey},
	},
	{
		Description: MRunningLoadtestCountStat.Description(),
		Measure:     MRunningLoadtestCountStat,
		Aggregation: view.LastValue(),
	},
}

// StatsReporter defines the interface for sending metrics
type StatsReporter interface {
	// ReportQueueDepth reports the queue depth metric
	ReportQueueDepth(v int64) error

	// ReportReconcile reports the count and latency metrics for a reconcile operation
	ReportReconcile(duration time.Duration, key, success string) error
}

// Reporter holds cached metric objects to report metrics
type reporter struct {
	reconciler string
	globalCtx  context.Context
}

// NewStatsReporter creates a reporter that collects and reports metrics
func NewStatsReporter(reconciler string) (StatsReporter, error) {
	// Reconciler tag is static. Create a context containing that and cache it.
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(reconcilerTagKey, reconciler))
	if err != nil {
		return nil, err
	}

	return &reporter{reconciler: reconciler, globalCtx: ctx}, nil
}

// ReportQueueDepth reports the queue depth metric
func (r *reporter) ReportQueueDepth(v int64) error {
	if r.globalCtx == nil {
		return errors.New("reporter is not initialized correctly")
	}
	stats.Record(r.globalCtx, workQueueDepthStat.M(v))
	return nil
}

// ReportReconcile reports the count and latency metrics for a reconcile operation
func (r *reporter) ReportReconcile(duration time.Duration, key, success string) error {
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(reconcilerTagKey, r.reconciler),
		tag.Insert(keyTagKey, key),
		tag.Insert(successTagKey, success))
	if err != nil {
		return err
	}

	stats.Record(ctx, reconcileCountStat.M(1))
	stats.Record(ctx, reconcileLatencyStat.M(int64(duration/time.Millisecond)))
	return nil
}

func mustNewTagKey(s string) tag.Key {
	tagKey, err := tag.NewKey(s)
	if err != nil {
		panic(err)
	}
	return tagKey
}
