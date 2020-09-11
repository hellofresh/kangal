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

//CreateLoadTestHandler creates loadtest CR on POST request
func CreateLoadTestHandler(loadTestClient loadTestV1.LoadTestInterface, maxLoadTestsRun int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := mPkg.GetLogger(r.Context())

		ctx, cancel := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
		defer cancel()

		// check the number of active loadtests currently running on the cluster
		activeLoadTests, err := loadtest.CountActiveLoadtests(ctx, loadTestClient)
		if err != nil {
			logger.Error("Could not count active load tests", zap.Error(err))
			render.Render(w, r, cHttp.ErrResponse(http.StatusInternalServerError, "Could not count active load tests"))
			return
		}

		if activeLoadTests >= maxLoadTestsRun {
			logger.Warn("number of active load tests reached limit", zap.Int("current", activeLoadTests), zap.Int("limit", maxLoadTestsRun))
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

		lto, err := loadtest.CreateLoadTestCR(ctx, loadTestClient, loadTest, logger)
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
	})
}

//FileToString converts file to string
func FileToString(f io.ReadCloser) (string, error) {
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

//ConvertTestName converts testfile name to valid LoadTest object name
//example: my_load_test.jmx to my-load-test
func ConvertTestName(n string) string {
	noSuffix := strings.TrimSuffix(n, ".jmx")

	tf := strings.ToLower(noSuffix)

	if strings.Contains(tf, "_") {
		nu := strings.ReplaceAll(tf, "_", "-")
		return nu
	}

	return tf
}

//DeleteLoadTestHandler deletes load test CR
func DeleteLoadTestHandler(loadTestClient loadTestV1.LoadTestInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := mPkg.GetLogger(r.Context())

		ctx, cancel := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
		defer cancel()

		ltID := chi.URLParam(r, loadTestID)
		logger.Debug("Deleting loadtest", zap.Any("ltID", ltID))

		err := loadtest.DeleteLoadTestCR(ctx, loadTestClient, ltID, logger)
		if err != nil {
			logger.Error("Could not delete load test with error:", zap.Error(err))
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}

		render.NoContent(w, r)
	})
}

//GetLoadTestHandler returns the loadtest CR info.
//For now it returns only created namespace,
//will be extended when more business logic added
func GetLoadTestHandler(loadTestClient loadTestV1.LoadTestInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := mPkg.GetLogger(r.Context())

		ctx, cancel := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
		defer cancel()

		ltID := chi.URLParam(r, loadTestID)
		logger.Debug("Retrieving info for loadtest", zap.Any("ltID", ltID))

		result, err := loadtest.GetLoadtestCR(ctx, loadTestClient, ltID, logger)
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
	})
}
