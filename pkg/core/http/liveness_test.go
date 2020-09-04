package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLivenessHandler(t *testing.T) {
	request, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()

	LivenessHandler("Kangal Proxy")(w, request)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
