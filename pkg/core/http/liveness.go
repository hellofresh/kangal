package http

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
)

//LivenessHandler returns ok response for liveness probe
func LivenessHandler(service string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, &Response{
			HTTPStatusCode: http.StatusOK,
			StatusText:     fmt.Sprintf("%s is running", service),
		})
	}
}
