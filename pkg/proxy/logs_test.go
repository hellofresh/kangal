package proxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
)

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}

	return cli, s.Close
}

const expectedResponse string = "Testing log resposne"

func TestDoRequest(t *testing.T) {
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	handler, s := testingHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedResponse))
	}))
	defer s()

	uri, _ := url.Parse("http://localhost/some/base/url/path")

	req := restclient.NewRequestWithClient(uri, "", restclient.ClientContentConfig{GroupVersion: schema.GroupVersion{Group: "test"}}, handler)

	response, err := doRequest(req)
	assert.Nil(t, err)
	assert.Equal(t, expectedResponse, string(response))

	// Test Error handler
	errHandler, s2 := testingHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer s2()

	req = restclient.NewRequestWithClient(uri, "", restclient.ClientContentConfig{GroupVersion: schema.GroupVersion{Group: "test"}}, errHandler)

	_, err = doRequest(req)
	assert.Error(t, err)
}
