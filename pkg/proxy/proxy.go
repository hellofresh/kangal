package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"go.uber.org/zap"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	restClient "k8s.io/client-go/rest"

	"github.com/hellofresh/kangal/pkg/backends"
	cHttp "github.com/hellofresh/kangal/pkg/core/http"
	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
	kube "github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	mimeJSON = "application/json; charset=utf-8"
)

// Proxy handler
type Proxy struct {
	maxLoadTestsRun     int
	maxListLimit        int64
	registry            backends.Registry
	kubeClient          *kube.Client
	allowedCustomImages bool
}

// MetricsReporter used to interface with the metrics configurations
type MetricsReporter struct {
	countRunningLoadtests metric.Int64ObservableUpDownCounter
}

// NewMetricsReporter contains loadtest metrics definition
func NewMetricsReporter(meter metric.Meter, kubeClient *kube.Client) (*MetricsReporter, error) {
	countRunningLoadtests, err := meter.Int64ObservableUpDownCounter(
		"kangal_loadtests_count",
		metric.WithDescription("Current number of loadtests in cluster, grouped by type and phase"),
		metric.WithInt64Callback(func(ctx context.Context, io metric.Int64Observer) error {
			states, types, err := kubeClient.CountExistingLoadtests()
			if err != nil {
				return fmt.Errorf("could not get metric data for CountExistingLoadtests: %w", err)
			}
			for k, v := range states {
				io.Observe(v, metric.WithAttributes(attribute.String("phase", k.String())))
			}

			for k, v := range types {
				io.Observe(v, metric.WithAttributes(attribute.String("type", k.String())))
			}
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("could not register countRunningLoadtests metric: %w", err)
	}

	return &MetricsReporter{
			countRunningLoadtests: countRunningLoadtests,
		},
		nil
}

// NewProxy returns new Proxy handlers
func NewProxy(maxLoadTestsRun int, registry backends.Registry, kubeClient *kube.Client, maxListLimit int64, allowedCustomImages bool) *Proxy {
	return &Proxy{
		maxLoadTestsRun:     maxLoadTestsRun,
		registry:            registry,
		kubeClient:          kubeClient,
		maxListLimit:        maxListLimit,
		allowedCustomImages: allowedCustomImages,
	}
}

// LoadTestStatusPage represents a page of load tests.
type LoadTestStatusPage struct {
	Limit    int64            `json:"limit"`
	Continue string           `json:"continue"`
	Remain   *int64           `json:"remain"`
	Items    []LoadTestStatus `json:"items"`
}

// LoadTestStatus defines response structure for status request
type LoadTestStatus struct {
	Type            string                      `json:"type"`
	DistributedPods int32                       `json:"distributedPods"`        // number of distributed pods requested
	Namespace       string                      `json:"loadtestName,omitempty"` // namespace created equals the loadtest name
	Phase           string                      `json:"phase,omitempty"`        // jmeter loadtest status
	Tags            apisLoadTestV1.LoadTestTags `json:"tags"`
	HasEnvVars      bool                        `json:"hasEnvVars"`
	HasTestData     bool                        `json:"hasTestData"`
}

// List lists all the load tests.
func (p *Proxy) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := mPkg.GetLogger(ctx)

	opt, err := fromHTTPRequestToListOptions(r, p.maxListLimit)
	if err != nil {
		logger.Error("could not parse filter", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))

		return
	}

	logger.Debug("Retrieving info for load tests")

	loadTests, err := p.kubeClient.ListLoadTest(ctx, *opt)
	if err != nil {
		logger.Error("could not list load tests", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusInternalServerError, err.Error()))

		return
	}

	items := make([]LoadTestStatus, len(loadTests.Items))
	for i, lt := range loadTests.Items {
		items[i] = LoadTestStatus{
			Type:            lt.Spec.Type.String(),
			DistributedPods: *lt.Spec.DistributedPods,
			Namespace:       lt.Status.Namespace,
			Phase:           lt.Status.Phase.String(),
			Tags:            lt.Spec.Tags,
			HasEnvVars:      len(lt.Spec.EnvVars) != 0,
			HasTestData:     len(lt.Spec.TestData) != 0,
		}
	}

	render.JSON(w, r, &LoadTestStatusPage{
		Limit:    opt.Limit,
		Continue: loadTests.Continue,
		Remain:   loadTests.RemainingItemCount,
		Items:    items,
	})
}

// Create creates loadtest CR on POST request
func (p *Proxy) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := mPkg.GetLogger(ctx)

	// Making valid LoadTestSpec based on HTTP request
	ltSpec, err := fromHTTPRequestToLoadTestSpec(r, logger, p.allowedCustomImages)
	if err != nil {
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	backend, err := p.registry.GetBackend(ltSpec.Type)
	if err != nil {
		logger.Error("could not get backend", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	err = backend.TransformLoadTestSpec(&ltSpec)
	if err != nil {
		logger.Error("could not transform LoadTest spec", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	// Building LoadTest based on specs
	loadTest, err := apisLoadTestV1.BuildLoadTestObject(ltSpec)
	if err != nil {
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	// Find the old load test with the same data
	labeledLoadTests, err := p.kubeClient.GetLoadTestsByLabel(ctx, loadTest)
	if err != nil {
		logger.Error("Could not count active load tests with given hash", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusInternalServerError, "Could not count active load tests with given hash"))
		return
	}

	if len(labeledLoadTests.Items) > 0 {
		if !loadTest.Spec.Overwrite {
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest,
				"Load test with given testfile already exists, aborting. Please delete existing load test and try again."))
			return
		}

		// If users wants to overwrite
		for _, item := range labeledLoadTests.Items {
			// Remove the old tests
			err := p.kubeClient.DeleteLoadTest(ctx, item.Name)
			if err != nil {
				logger.Error("Could not delete load test with error", zap.Error(err))
				render.Render(w, r, cHttp.ErrResponse(http.StatusConflict, err.Error()))
				return
			}
		}
	}

	// check the number of active loadtests currently running on the cluster
	testsByPhase, _, err := p.kubeClient.CountExistingLoadtests()
	if err != nil {
		logger.Error("Could not count active load tests", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusInternalServerError, "Could not count active load tests"))
		return
	}
	activeLoadTests := testsByPhase["running"] + testsByPhase["creating"]

	if int(activeLoadTests) >= p.maxLoadTestsRun {
		logger.Warn("number of active load tests reached limit", zap.Int("current", int(activeLoadTests)), zap.Int("limit", p.maxLoadTestsRun))
		render.Render(w, r, cHttp.ErrResponse(http.StatusTooManyRequests, "Number of active load tests reached limit"))
		return
	}

	// Pushing LoadTest to Kubernetes
	loadTestName, err := p.kubeClient.CreateLoadTest(ctx, loadTest)
	if err != nil {
		logger.Error("Could not create load test", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusConflict, err.Error()))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, &LoadTestStatus{
		Type:            loadTest.Spec.Type.String(),
		DistributedPods: *loadTest.Spec.DistributedPods,
		Namespace:       loadTestName,
		Phase:           string(apisLoadTestV1.LoadTestCreating),
		Tags:            loadTest.Spec.Tags,
		HasEnvVars:      len(loadTest.Spec.EnvVars) != 0,
		HasTestData:     len(loadTest.Spec.TestData) != 0,
	})
}

// Delete deletes load test CR
func (p *Proxy) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := mPkg.GetLogger(ctx)

	ltID := chi.URLParam(r, loadTestID)
	logger.Debug("Deleting loadtest", zap.String("ltID", ltID))

	err := p.kubeClient.DeleteLoadTest(ctx, ltID)
	if err != nil {
		logger.Error("Could not delete load test with error", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	render.NoContent(w, r)
}

// Get returns the loadtest CR info
func (p *Proxy) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := mPkg.GetLogger(ctx)

	ltID := chi.URLParam(r, loadTestID)
	logger.Debug("Retrieving info for loadtest", zap.String("ltID", ltID))

	result, err := p.kubeClient.GetLoadTest(ctx, ltID)
	if err != nil {
		logger.Error("Could not get load test info with error", zap.Error(err))

		if k8sAPIErrors.IsNotFound(err) {
			render.Render(w, r, cHttp.ErrResponse(http.StatusNotFound, err.Error()))
			return
		}

		render.Render(w, r, cHttp.ErrResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	render.JSON(w, r, &LoadTestStatus{
		Type:            result.Spec.Type.String(),
		DistributedPods: *result.Spec.DistributedPods,
		Namespace:       result.Status.Namespace,
		Phase:           result.Status.Phase.String(),
		Tags:            result.Spec.Tags,
		HasEnvVars:      len(result.Spec.EnvVars) != 0,
		HasTestData:     len(result.Spec.TestData) != 0,
	})
}

// GetLogs returns the loadtest logs from master or worker pods
func (p *Proxy) GetLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := mPkg.GetLogger(ctx)
	ltID := chi.URLParam(r, loadTestID)
	workerID := chi.URLParam(r, workerPodID)
	var logsRequest *restClient.Request
	logger.Info("Retrieving logs for loadtest", zap.String("ltID", ltID))

	loadTest, err := p.kubeClient.GetLoadTest(ctx, ltID)
	if err != nil {
		logger.Error("Could not get load test info with error", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	namespace := loadTest.Status.Namespace
	// if no namespace was created we can not get logs
	if namespace == "" {
		render.Render(w, r, cHttp.ErrResponse(http.StatusNoContent, "no logs found in load test resources"))
		return
	}

	if workerID == "" {
		logger.Info("Returning master pod logs")
		logsRequest, err = p.kubeClient.GetMasterPodRequest(ctx, namespace)
		if err != nil {
			logger.Error("Could not get load test logs request:", zap.Error(err))
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}
	} else {
		logger.Info("Returning worker pod logs")
		logsRequest, err = p.kubeClient.GetWorkerPodRequest(ctx, namespace, workerID)
		if err != nil {
			logger.Error("Could not get load test logs request:", zap.Error(err))
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}
	}

	logs, err := doRequest(logsRequest)
	if err != nil {
		logger.Error("Could not get load test logs:", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	io.WriteString(w, string(logs))
}

func doRequest(req *restClient.Request) ([]byte, error) {
	stream, err := req.Stream(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error in opening stream: %w", err)
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return nil, fmt.Errorf("error in copy information from podLogs to buf: %w", err)
	}

	return buf.Bytes(), nil
}
