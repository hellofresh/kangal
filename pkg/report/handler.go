package report

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	khttp "github.com/hellofresh/kangal/pkg/core/http"
	kk8s "github.com/hellofresh/kangal/pkg/kubernetes"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/minio/minio-go/v6"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

//ShowHandler method returns response from file bucket in defined object storage
func ShowHandler() func(w http.ResponseWriter, r *http.Request) {
	if minioClient == nil {
		panic("client was not initialized, please initialize object storage client")
	}
	if bucketName == "" {
		panic("bucket name was not defined or empty")
	}

	tmpDir := fmt.Sprintf("%s/kangal", strings.TrimRight(os.TempDir(), "/"))

	go func() {
		ticker := time.NewTicker(5 * time.Hour)
		for range ticker.C {
			os.RemoveAll(tmpDir)
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		loadTestName := chi.URLParam(r, "id")
		file := chi.URLParam(r, "*")

		r.URL.Path = fmt.Sprintf("/%s", loadTestName)
		if file != "" {
			r.URL.Path += fmt.Sprintf("/%s", file)
		}

		// first, try to handle uncompressed tar archive
		obj, err := minioClient.GetObject(bucketName, loadTestName, minio.GetObjectOptions{})
		if nil != err {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		objStat, err := obj.Stat()
		if nil == err && "application/x-tar" == objStat.ContentType {
			prefix := fmt.Sprintf("%s/%s", tmpDir, loadTestName)
			err = untar(prefix, obj, afero.NewOsFs())
			if nil != err {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			r.URL.Path = fmt.Sprintf("/%s", file)
			http.FileServer(http.Dir(prefix)).ServeHTTP(w, r)
			return
		}

		http.FileServer(fs).ServeHTTP(w, r)
	}
}

func untar(prefix string, obj io.Reader, afs afero.Fs) error {
	_, err := afs.Stat(prefix)
	if nil == err {
		// means its already exists locally
		return nil
	}

	reader := tar.NewReader(obj)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if nil != err {
			return err
		}
		if header == nil {
			continue
		}
		target := fmt.Sprintf("%s/%s", prefix, strings.Trim(header.Name, "./"))
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := afs.Stat(target); err != nil {
				if err := afs.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := afs.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, reader); err != nil {
				return err
			}
			f.Close()
		}
	}
}

//PersistHandler method streams request to storage presigned URL
func PersistHandler(kubeClient *kk8s.Client, logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	if minioClient == nil {
		panic("client was not initialized, please initialize object storage client")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		loadTestName := chi.URLParam(r, "id")

		_, err := kubeClient.GetLoadTest(r.Context(), loadTestName)
		if errors.IsNotFound(err) {
			render.Render(w, r, khttp.ErrResponse(http.StatusNotFound, err.Error()))
			return
		}
		if err != nil {
			render.Render(w, r, khttp.ErrResponse(http.StatusInternalServerError, err.Error()))
			return
		}

		url, err := newPreSignedPutURL(loadTestName)
		if nil != err {
			render.Render(w, r, khttp.ErrResponse(http.StatusInternalServerError, err.Error()))
			return
		}

		proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, url.String(), r.Body)
		if nil != err {
			render.Render(w, r, khttp.ErrResponse(http.StatusInternalServerError, err.Error()))
			return
		}
		proxyReq.ContentLength = r.ContentLength
		proxyReq.Header = r.Header

		proxyResp, err := httpClient.Do(proxyReq)
		if nil != err {
			logger.Error("Failed to persist report", zap.Error(err), zap.String("loadtest", loadTestName))
			render.Render(w, r, khttp.ErrResponse(http.StatusInternalServerError, err.Error()))
			return
		}
		defer proxyResp.Body.Close()

		body := "Report persisted"

		if http.StatusOK != proxyResp.StatusCode {
			b, _ := ioutil.ReadAll(proxyResp.Body)
			body = string(b)
		}

		render.Status(r, proxyResp.StatusCode)
		render.JSON(w, r, body)
	}
}
