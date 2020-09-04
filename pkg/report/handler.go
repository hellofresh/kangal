package report

import (
	"net/http"
	"strings"
)

//ShowHandler method returns response from file bucket in defined object storage
func ShowHandler() func(w http.ResponseWriter, r *http.Request) {
	if minioClient == nil {
		panic("client was not initialized, please initialize object storage client")
	}
	if bucketName == "" {
		panic("bucket name was not defined or empty")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		h := http.FileServer(fs)
		// Make path from the /load-test/loadtest-name/report/ -> /loadtest-name format
		r.URL.Path = strings.Replace(r.URL.Path, "load-test/", "", -1)
		r.URL.Path = strings.Replace(r.URL.Path, "/report", "", -1)
		h.ServeHTTP(w, r)
	}
}
