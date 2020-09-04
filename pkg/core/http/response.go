package http

import (
	"net/http"

	"github.com/go-chi/render"
)

// Response defines HTTP response structure
type Response struct {
	HTTPStatusCode int    `json:"-"`                // http response status code
	StatusText     string `json:"status,omitempty"` // user-level status message
	ErrorText      string `json:"error,omitempty"`  // application-level error message, for debugging
}

// ErrResponse returns Response struct with provided HTTP status code and error text
func ErrResponse(status int, err string) *Response {
	return &Response{
		HTTPStatusCode: status,
		ErrorText:      err,
	}
}

// Render renders a response
func (e *Response) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}
