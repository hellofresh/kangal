package report

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi"
	kk8s "github.com/hellofresh/kangal/pkg/kubernetes"
	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
	"github.com/minio/minio-go/v6"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// fakeFS mocks http.FileSystem
type fakeFS struct{}

func (fakeFS) Open(name string) (http.File, error) {
	if name != "/loadtest-name" {
		return nil, errors.New("path is incorrect")
	}
	return os.Open("handler_test.go")
}

func TestShowHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/load-test/loadtest-name/report/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// init dependencies for report package
	minioClient, _ = minio.New("localhost:80", "access-key", "secret-access-key", false)
	bucketName = "bucket-name"
	fs = &fakeFS{}

	rr := httptest.NewRecorder()

	handler := chi.NewRouter()
	handler.Get("/load-test/{id}/report/*", ShowHandler())

	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestPersistHandler(t *testing.T) {
	var scenarios = []struct {
		fakeResponseStatusCode int
		getLoadTestsFn         func(action k8stesting.Action) (handled bool, ret runtime.Object, err error)
		expectedStatusCode     int
	}{
		{
			fakeResponseStatusCode: http.StatusOK,
			getLoadTestsFn: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &loadtestV1.LoadTest{}, nil
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			fakeResponseStatusCode: http.StatusOK,
			getLoadTestsFn: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(loadtestV1.Resource("loadtests"), "loadtest-name")
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			fakeResponseStatusCode: http.StatusUnauthorized,
			getLoadTestsFn: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &loadtestV1.LoadTest{}, nil
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
	}

	req, err := http.NewRequest("PUT", "/load-test/loadtest-name/report", nil)
	if err != nil {
		t.Fatal(err)
	}

	// init dependencies for report package
	minioClient, _ = minio.NewWithRegion("localhost:80", "access-key", "secret-access-key", false, "us-east1")
	bucketName = "bucket-name"
	expires = time.Second

	for _, scenario := range scenarios {
		httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: scenario.fakeResponseStatusCode,
				// Must be set to non-nil value or it panics
				Header: make(http.Header),
			}
		})}

		kangalFakeClientSet := fake.NewSimpleClientset()
		kangalFakeClientSet.PrependReactor("get", "loadtests", scenario.getLoadTestsFn)

		kangalKubeClient := kk8s.NewClient(
			kangalFakeClientSet.KangalV1().LoadTests(),
			k8sfake.NewSimpleClientset(),
			zap.NewNop(),
		)

		rr := httptest.NewRecorder()
		handler := chi.NewRouter()
		handler.Put("/load-test/{id}/report", PersistHandler(kangalKubeClient))
		handler.ServeHTTP(rr, req)

		assert.Equal(t, rr.Code, scenario.expectedStatusCode)
	}
}
