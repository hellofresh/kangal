package report

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi"
	"github.com/minio/minio-go/v6"
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
