package report

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/minio/minio-go/v6"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	kk8s "github.com/hellofresh/kangal/pkg/kubernetes"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	req, err := http.NewRequest("GET", "/load-test/loadtest-name/report/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// init dependencies for report package
	minioClient, _ = minio.New(srv.Listener.Addr().String(), "access-key", "secret-access-key", false)
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
		name                   string
		fakeResponseStatusCode int
		getLoadTestsFn         func(action k8stesting.Action) (handled bool, ret runtime.Object, err error)
		expectedStatusCode     int
	}{
		{
			name:                   "All good",
			fakeResponseStatusCode: http.StatusOK,
			getLoadTestsFn: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &loadTestV1.LoadTest{}, nil
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:                   "LoadTest not found",
			fakeResponseStatusCode: http.StatusOK,
			getLoadTestsFn: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(loadTestV1.Resource("loadtests"), "loadtest-name")
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:                   "S3 wrong credentials",
			fakeResponseStatusCode: http.StatusUnauthorized,
			getLoadTestsFn: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &loadTestV1.LoadTest{}, nil
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:                   "Request timed out",
			fakeResponseStatusCode: http.StatusRequestTimeout,
			getLoadTestsFn: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &loadTestV1.LoadTest{}, nil
			},
			expectedStatusCode: http.StatusRequestTimeout,
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
	logger := zap.NewNop()

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
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
			handler.Put("/load-test/{id}/report", PersistHandler(kangalKubeClient, logger))
			handler.ServeHTTP(rr, req)

			assert.Equal(t, rr.Code, scenario.expectedStatusCode)
		})
	}
}

func TestUntar(t *testing.T) {
	tarball := bytes.NewBuffer(nil)

	writer := tar.NewWriter(tarball)
	defer writer.Close()

	for i := 0; i < 10; i++ {
		str := fmt.Sprintf("my-file-%d", i)
		file := bytes.NewReader([]byte(str))

		header := &tar.Header{
			Name:    str,
			Size:    file.Size(),
			Mode:    0644,
			ModTime: time.Now(),
		}

		err := writer.WriteHeader(header)
		if err != nil {
			t.Fatal(err)
		}

		_, err = io.Copy(writer, file)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := untar("/my-load-test", tarball, afero.NewMemMapFs())
	assert.NoError(t, err)
}
