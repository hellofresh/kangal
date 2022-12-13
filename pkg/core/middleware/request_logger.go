package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	"go.uber.org/zap"
)

// RequestLogger is a struct for logging request
type RequestLogger struct{}

// NewRequestLogger creates new request logger
func NewRequestLogger() *RequestLogger {
	return &RequestLogger{}
}

// Handler handles request logging
func (m *RequestLogger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GetLogger(r.Context()).Debug("Started request", zap.String("method", r.Method), zap.String("path", r.URL.Path))

		m := httpsnoop.CaptureMetrics(next, w, r)

		logEntry := GetLogger(r.Context()).With(
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("host", r.Host),
			zap.String("request", r.RequestURI),
			zap.String("remote-addr", r.RemoteAddr),
			zap.String("referer", r.Referer()),
			zap.String("user-agent", r.UserAgent()),
			zap.Int("code", m.Code),
			zap.Int("duration-ms", int(m.Duration/time.Millisecond)),
			zap.String("duration-fmt", m.Duration.String()),
		)

		if IsStatusRequest(r) {
			logEntry.Debug("Finished serving request")
			return
		}

		logEntry.Info("Finished serving request")
	})
}

// IsStatusRequest checks if the request is a health check request
func IsStatusRequest(r *http.Request) bool {
	return strings.HasSuffix(r.URL.Path, "status")
}
