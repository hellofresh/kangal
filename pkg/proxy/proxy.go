package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"go.uber.org/zap"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	kube "k8s.io/client-go/kubernetes"
	restClient "k8s.io/client-go/rest"

	loadtest "github.com/hellofresh/kangal/pkg/controller"
	cHttp "github.com/hellofresh/kangal/pkg/core/http"
	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/typed/loadtest/v1"
)

const (
	mimeJSON = "application/json; charset=utf-8"
)

var (
	// ErrFileToStringEmpty is the error returned when the defined users file is empty
	ErrFileToStringEmpty error = errors.New("file is empty")
)

// Proxy handler
type Proxy struct {
	maxLoadTestsRun int
	loadTestClient  loadTestV1.LoadTestInterface
	kubeClient      kube.Interface
}

// NewProxy returns new Proxy handlers
func NewProxy(maxLoadTestsRun int, loadTestClient loadTestV1.LoadTestInterface, kubeClient kube.Interface) *Proxy {
	return &Proxy{
		maxLoadTestsRun: maxLoadTestsRun,
		loadTestClient:  loadTestClient,
		kubeClient:      kubeClient,
	}
}

// LoadTestStatus defines response structure for status request
type LoadTestStatus struct {
	Type            string `json:"type"`
	DistributedPods int32  `json:"distributedPods"`        // number of distributed pods requested
	Namespace       string `json:"loadtestName,omitempty"` // namespace created equals the loadtest name
	Phase           string `json:"phase,omitempty"`        // jmeter loadtest status
	HasEnvVars      bool   `json:"hasEnvVars"`
	HasTestData     bool   `json:"hasTestData"`
}

func getLoadTestType(r *http.Request) apisLoadTestV1.LoadTestType {
	return apisLoadTestV1.LoadTestType(r.FormValue(backendType))
}

//Create creates loadtest CR on POST request
func (p *Proxy) Create(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())

	ctx, cancel := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
	defer cancel()

	// check the number of active loadtests currently running on the cluster
	activeLoadTests, err := loadtest.CountActiveLoadtests(ctx, p.loadTestClient)
	if err != nil {
		logger.Error("Could not count active load tests", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusInternalServerError, "Could not count active load tests"))
		return
	}

	if activeLoadTests >= p.maxLoadTestsRun {
		logger.Warn("number of active load tests reached limit", zap.Int("current", activeLoadTests), zap.Int("limit", p.maxLoadTestsRun))
		render.Render(w, r, cHttp.ErrResponse(http.StatusTooManyRequests, "Number of active load tests reached limit"))
		return
	}
	var loadTest *apisLoadTestV1.LoadTest
	switch ltType := getLoadTestType(r); ltType {
	case apisLoadTestV1.LoadTestTypeJMeter, apisLoadTestV1.LoadTestTypeFake:
		jmSpec, err := FromHTTPRequestToJMeter(r, ltType, logger)
		if err != nil {
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}

		jm, err := NewJMeterLoadTest(jmSpec, logger)
		if err != nil {
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}

		loadTest = jm.ToLoadTest()
	default:
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, fmt.Sprintf("loadtest %q %q is not supported", backendType, ltType)))
		return
	}

	lto, err := loadtest.CreateLoadTestCR(ctx, p.loadTestClient, loadTest, logger)
	if err != nil {
		if err == os.ErrExist {
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest,
				"Load test with given testfile already exists, aborting. Please delete existing load test and try again."))
			return
		}

		logger.Error("Could not create load test", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusConflict, err.Error()))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, &LoadTestStatus{
		Type:            string(loadTest.Spec.Type),
		DistributedPods: *loadTest.Spec.DistributedPods,
		Namespace:       lto,
		Phase:           string(apisLoadTestV1.LoadTestCreating),
		HasEnvVars:      len(loadTest.Spec.EnvVars) != 0,
		HasTestData:     loadTest.Spec.TestData != "",
	})
}

//Delete deletes load test CR
func (p *Proxy) Delete(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())

	ctx, cancel := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
	defer cancel()

	ltID := chi.URLParam(r, loadTestID)
	logger.Debug("Deleting loadtest", zap.String("ltID", ltID))

	err := loadtest.DeleteLoadTestCR(ctx, p.loadTestClient, ltID, logger)
	if err != nil {
		logger.Error("Could not delete load test with error:", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	render.NoContent(w, r)
}

//Get returns the loadtest CR info
func (p *Proxy) Get(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())

	ctx, cancel := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
	defer cancel()

	ltID := chi.URLParam(r, loadTestID)
	logger.Debug("Retrieving info for loadtest", zap.Any("ltID", ltID))

	result, err := loadtest.GetLoadtestCR(ctx, p.loadTestClient, ltID, logger)
	if err != nil {
		logger.Error("Could not get load test info with error:", zap.Error(err))

		if k8sErr, ok := err.(k8sAPIErrors.APIStatus); ok && k8sErr.Status().Code == http.StatusNotFound {
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
		HasEnvVars:      len(result.Spec.EnvVars) != 0,
		HasTestData:     len(result.Spec.TestData) != 0,
	})
}

//GetLogs returns the loadtest CR info
func (p *Proxy) GetLogs(w http.ResponseWriter, r *http.Request) {
	logger := mPkg.GetLogger(r.Context())
	ltID := chi.URLParam(r, loadTestID)
	logger.Info("Retrieving info for loadtest", zap.String("ltID", ltID))

	ctx, cancel := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
	defer cancel()
	loadTest, err := loadtest.GetLoadtestCR(ctx, p.loadTestClient, ltID, logger)
	if err != nil {
		logger.Error("Could not get load test info with error:", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
	}

	namespace := loadTest.Status.Namespace
	// if no namespace was created we can not get logs
	if namespace == "" {
		render.Render(w, r, cHttp.ErrResponse(http.StatusNoContent, "no logs found in load test resources"))
		return
	}

	ctxJMeterLogs, cancelJMeterLogs := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
	defer cancelJMeterLogs()
	logsRequest, err := loadtest.GetMasterPodLogs(ctxJMeterLogs, p.kubeClient, namespace, logger)
	if err != nil {
		logger.Error("Could not get load test logs request:", zap.Error(err))
		render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
		return
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

//fileToString converts file to string
func fileToString(f io.ReadCloser) (string, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(f); err != nil {
		return "", err
	}

	defer f.Close()
	s := buf.String()

	if len(s) == 0 {
		return "", ErrFileToStringEmpty
	}

	return s, nil
}

//convertTestName converts testfile name to valid LoadTest object name
//example: my_load_test.jmx to my-load-test
func convertTestName(n string) string {
	noSuffix := strings.TrimSuffix(n, ".jmx")

	tf := strings.ToLower(noSuffix)

	if strings.Contains(tf, "_") {
		nu := strings.ReplaceAll(tf, "_", "-")
		return nu
	}

	return tf
}

func doRequest(req *restClient.Request) ([]byte, error) {
	stream, err := req.Stream(context.Background())
	if err != nil {
		return []byte{}, fmt.Errorf("error in opening stream: %w", err)
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return []byte{}, errors.New("error in copy information from podLogs to buf")
	}

	return buf.Bytes(), nil
}
