package proxy

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
)

func TestOpenAPISpecHandler_static(t *testing.T) {
	rawSpec, err := ioutil.ReadFile("../../openapi.json")
	require.NoError(t, err)

	cfg := OpenAPIConfig{
		SpecPath: "../../",
		SpecFile: "openapi.json",
	}

	staticHandler := OpenAPISpecHandler(cfg)

	rq, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	logger := zap.NewNop()
	ctx := mPkg.SetLogger(context.Background(), logger)

	w := httptest.NewRecorder()

	staticHandler(w, rq.WithContext(ctx))
	rs := w.Result()

	rsBody, err := ioutil.ReadAll(rs.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rs.StatusCode)
	assert.Equal(t, rawSpec, rsBody)
}

func TestOpenAPISpecHandler_custom(t *testing.T) {
	cfg := OpenAPIConfig{
		SpecPath:          "../../",
		SpecFile:          "openapi.json",
		ServerURL:         "https://example.com",
		ServerDescription: "Such description, much descriptive",
	}

	staticHandler := OpenAPISpecHandler(cfg)

	rq, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	logger := zap.NewNop()
	ctx := mPkg.SetLogger(context.Background(), logger)

	w := httptest.NewRecorder()

	staticHandler(w, rq.WithContext(ctx))
	rs := w.Result()

	rsBody, err := ioutil.ReadAll(rs.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rs.StatusCode)
	assert.Contains(t, string(rsBody), cfg.ServerURL)
	assert.Contains(t, string(rsBody), cfg.ServerDescription)
}

func TestOpenAPIUIHandler_redirect(t *testing.T) {
	cfg := OpenAPIConfig{
		UIUrl: "https://openapi.example.com/",
	}

	uiHandler := OpenAPIUIHandler(cfg)

	rq, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	uiHandler(w, rq)
	rs := w.Result()

	assert.Equal(t, http.StatusFound, rs.StatusCode)
	assert.Equal(t, cfg.UIUrl, rs.Header.Get("Location"))
}

func TestOpenAPIUIHandler_no_ui(t *testing.T) {
	cfg := OpenAPIConfig{}

	uiHandler := OpenAPIUIHandler(cfg)

	rq, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	logger := zap.NewNop()
	ctx := mPkg.SetLogger(context.Background(), logger)

	w := httptest.NewRecorder()

	uiHandler(w, rq.WithContext(ctx))
	rs := w.Result()

	assert.Equal(t, http.StatusNotFound, rs.StatusCode)
}
