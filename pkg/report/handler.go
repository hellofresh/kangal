package report

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-chi/chi"

	kube "github.com/hellofresh/kangal/pkg/kubernetes"
)

var httpClient = &http.Client{}

//ShowHandler method returns response from file bucket in defined object storage
func ShowHandler() func(w http.ResponseWriter, r *http.Request) {
	if minioClient == nil {
		panic("client was not initialized, please initialize object storage client")
	}
	if bucketName == "" {
		panic("bucket name was not defined or empty")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		loadTestName := chi.URLParam(r, "id")
		file := chi.URLParam(r, "*")

		r.URL.Path = fmt.Sprintf("/%s", loadTestName)
		if file != "" {
			r.URL.Path += fmt.Sprintf("/%s", file)
		}

		http.FileServer(fs).ServeHTTP(w, r)
	}
}

//PersistHandler method streams request to storage presigned URL
func PersistHandler(kubeClient *kube.Client) func(w http.ResponseWriter, r *http.Request) {
	if minioClient == nil {
		panic("client was not initialized, please initialize object storage client")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		loadTestName := chi.URLParam(r, "id")

		_, err := kubeClient.GetLoadTest(r.Context(), loadTestName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		url, err := newPreSignedPutURL(loadTestName)
		if nil != err {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		proxyReq, err := http.NewRequest(r.Method, url.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for header, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(header, value)
			}
		}
		proxyReq.ContentLength = r.ContentLength

		proxyResp, err := httpClient.Do(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		proxyRespBody, err := ioutil.ReadAll(proxyResp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(proxyResp.StatusCode)
		w.Write(proxyRespBody)
	}
}
