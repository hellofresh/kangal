package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"go.uber.org/zap"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	restClient "k8s.io/client-go/rest"

	"github.com/hellofresh/kangal/pkg/backends"
	loadtest "github.com/hellofresh/kangal/pkg/controller"
	cHttp "github.com/hellofresh/kangal/pkg/core/http"
	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
	kube "github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	mimeJSON = "application/json; charset=utf-8"
)

var (
	// ErrFileToStringEmpty is the error returned when the defined users file is empty
	ErrFileToStringEmpty error = errors.New("file is empty")
)

type loadTestSpecCreator func(*http.Request, backends.Config, *zap.Logger) (apisLoadTestV1.LoadTestSpec, error)

// Proxy handler
type Proxy struct {
	config              Config
	kubeClient          *kube.Client
	httpToSpecConverter loadTestSpecCreator
}

// NewProxy returns new Proxy handlers
func NewProxy(cfg Config, kubeClient *kube.Client, specCreator loadTestSpecCreator) *Proxy {
	return &Proxy{
		config:              cfg,
		kubeClient:          kubeClient,
		httpToSpecConverter: specCreator,
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

func getLoadTestType(r *http.Request) apisLoadTestV1.LoadTestType {
	return apisLoadTestV1.LoadTestType(r.FormValue(backendType))
}

// List lists all the load tests.
func (p *Proxy) List(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), loadtest.KubeTimeout)
	defer cancel()

	opt, err := fromHTTPRequestToListOptions(r)
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

	if opt.Phase != "" {
		loadTests = p.kubeClient.ListLoadTestsByPhase(loadTests, opt.Phase)
	}

	items := make([]LoadTestStatus, len(loadTests.Items))

	for i, lt := range loadTests.Items {
		items[i] = LoadTestStatus{
			Type:            string(lt.Spec.Type),
			DistributedPods: *lt.Spec.DistributedPods,
			Namespace:       lt.Status.Namespace,
			Phase:           string(lt.Status.Phase),
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

//Create creates loadtest CR on POST request
func (p *Proxy) Create(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), loadtest.KubeTimeout)
	defer cancel()

	// Making valid LoadTestSpec based on HTTP request
	ltSpec, err := p.httpToSpecConverter(r, p.config.Backends, logger)
	if err != nil {
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

		// If users wants to overwrite
		if loadTest.Spec.Overwrite == true {
			for _, item := range labeledLoadTests.Items {

				// Remove the old tests
				err := p.kubeClient.DeleteLoadTest(ctx, item.Name)
				if err != nil {
					logger.Error("Could not delete load test with error", zap.Error(err))
					render.Render(w, r, cHttp.ErrResponse(http.StatusConflict, err.Error()))
					return
				}
			}
		} else {
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest,
				"Load test with given testfile already exists, aborting. Please delete existing load test and try again."))
			return
		}
	}

	// check the number of active loadtests currently running on the cluster
	activeLoadTests, err := p.kubeClient.CountActiveLoadTests(ctx)
	if err != nil {
		logger.Error("Could not count active load tests", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusInternalServerError, "Could not count active load tests"))
		return
	}

	if activeLoadTests >= p.config.MaxLoadTestsRun {
		logger.Warn("number of active load tests reached limit", zap.Int("current", activeLoadTests), zap.Int("limit", p.config.MaxLoadTestsRun))
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
		Type:            string(loadTest.Spec.Type),
		DistributedPods: *loadTest.Spec.DistributedPods,
		Namespace:       loadTestName,
		Phase:           string(apisLoadTestV1.LoadTestCreating),
		Tags:            loadTest.Spec.Tags,
		HasEnvVars:      len(loadTest.Spec.EnvVars) != 0,
		HasTestData:     loadTest.Spec.TestData != "",
	})
}

//Delete deletes load test CR
func (p *Proxy) Delete(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), loadtest.KubeTimeout)
	defer cancel()

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

//Get returns the loadtest CR info
func (p *Proxy) Get(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), loadtest.KubeTimeout)
	defer cancel()

	ltID := chi.URLParam(r, loadTestID)
	logger.Debug("Retrieving info for loadtest", zap.String("ltID", ltID))

	result, err := p.kubeClient.GetLoadTest(ctx, ltID)
	if err != nil {
		logger.Error("Could not get load test info with error:", zap.Error(err))

		if k8sAPIErrors.IsNotFound(err) {
			render.Render(w, r, cHttp.ErrResponse(http.StatusNotFound, err.Error()))
			return
		}

		render.Render(w, r, cHttp.ErrResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	render.JSON(w, r, &LoadTestStatus{
		Type:            string(result.Spec.Type),
		DistributedPods: *result.Spec.DistributedPods,
		Namespace:       result.Status.Namespace,
		Phase:           string(result.Status.Phase),
		Tags:            result.Spec.Tags,
		HasEnvVars:      len(result.Spec.EnvVars) != 0,
		HasTestData:     len(result.Spec.TestData) != 0,
	})
}

//GetLogs returns the loadtest logs from master or worker pods
func (p *Proxy) GetLogs(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())
	ltID := chi.URLParam(r, loadTestID)
	workerID := chi.URLParam(r, workerPodID)
	var logsRequest *restClient.Request
	logger.Info("Retrieving logs for loadtest", zap.String("ltID", ltID))

	ctx, cancel := context.WithTimeout(r.Context(), loadtest.KubeTimeout)
	defer cancel()

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

	ctxLogs, cancelLogs := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
	defer cancelLogs()
	if workerID == "" {
		logger.Info("Returning master pod logs")
		logsRequest, err = p.kubeClient.GetMasterPodRequest(ctxLogs, namespace)
		if err != nil {
			logger.Error("Could not get load test logs request:", zap.Error(err))
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}
	} else {
		logger.Info("Returning worker pod logs")
		logsRequest, err = p.kubeClient.GetWorkerPodRequest(ctxLogs, namespace, workerID)
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
	return
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
