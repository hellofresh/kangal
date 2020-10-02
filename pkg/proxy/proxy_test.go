package proxy

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	"bou.ke/monkey"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
	kube "github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	fakeClientset "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
)

const shortDuration = 1 * time.Millisecond // a reasonable duration to block in an example
var (
	loadTest          = &apisLoadTestV1.LoadTest{}
	kubeClientSet     = fake.NewSimpleClientset()
	loadtestClientSet = fakeClientset.NewSimpleClientset()
)

func TestHTTPValidator(t *testing.T) {
	for _, tt := range []struct {
		name             string
		distributedPods  string
		failingLine      string
		loadTestType     string
		requestFiles     map[string]string
		expectedResponse string
	}{
		{
			"Valid JMeter",
			"1",
			"",
			"JMeter",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			"",
		},
		{
			"Valid Fake",
			"1",
			"",
			"Fake",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			"",
		},
		{
			"Empty distributed pods",
			"0",
			"distributedPods",
			"Fake",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			"The distributedPods field value can not be less than 1",
		},
		{
			"Invalid test file",
			"1",
			"testFile",
			"JMeter",
			map[string]string{
				"testFile": "testdata/valid/testdata.csv",
			},
			"The testFile field file extension csv is invalid",
		},
		{
			"Invalid envVars file",
			"1",
			"envVars",
			"JMeter",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
				"envVars":  "testdata/valid/loadtest.jmx",
			},
			"The envVars field file extension jmx is invalid",
		},
		{
			"Invalid testData file",
			"1",
			"testData",
			"JMeter",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
				"testData": "testdata/valid/loadtest.jmx",
			},
			"The testData field file extension jmx is invalid",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request, err := buildMocFormReq(tt.requestFiles, tt.distributedPods, tt.loadTestType)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			result := httpValidator(request)
			assert.Equal(t, tt.expectedResponse, result.Get(tt.failingLine))
		})
	}
}

func TestCreateWithTimeout(t *testing.T) {
	for _, tt := range []struct {
		name             string
		distributedPods  string
		failingLine      string
		loadTestType     string
		requestFiles     map[string]string
		expectedResponse string
	}{
		{
			"Valid JMeter",
			"1",
			"",
			"JMeter",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			"context deadline exceeded",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request, err := buildMocFormReq(tt.requestFiles, tt.distributedPods, tt.loadTestType)

			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			// Pass a context with a timeout to tell a blocking function that it
			// should abandon its work after the timeout elapses.
			ctx, cancel := context.WithTimeout(request.Context(), shortDuration)
			defer cancel()

			// Wait for tests to hit
			time.Sleep(1 * time.Millisecond)

			select {
			case <-time.After(1 * time.Second):
				t.Error("Expected to have a timeout error")
			case <-ctx.Done():
				assert.Equal(t, tt.expectedResponse, ctx.Err().Error())
			}

		})
	}
}

func TestProxyCreate(t *testing.T) {
	for _, tt := range []struct {
		name                string
		distributedPods     int
		failingLine         string
		loadTestType        apisLoadTestV1.LoadTestType
		requestFiles        map[string]string
		expectedCode        int
		expectedResponse    string
		expectedContentType string
	}{
		{
			"Valid request, only test file",
			10,
			"",
			apisLoadTestV1.LoadTestTypeJMeter,
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			http.StatusCreated,
			"{\"type\":\"JMeter\",\"distributedPods\":10,\"phase\":\"creating\",\"hasEnvVars\":false,\"hasTestData\":false}\n",
			"application/json; charset=utf-8",
		},
		{
			"Valid request, all files",
			10,
			"",
			apisLoadTestV1.LoadTestTypeFake,
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
				"testData": "testdata/valid/testdata.csv",
				"envVars":  "testdata/valid/envvars.csv",
			},
			http.StatusCreated,
			"{\"type\":\"Fake\",\"distributedPods\":10,\"phase\":\"creating\",\"hasEnvVars\":true,\"hasTestData\":true}\n",
			"application/json; charset=utf-8",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var logger = zaptest.NewLogger(t)

			loadtestClientSet.Fake.PrependReactor("create", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, loadTest, nil
			})
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			testProxyHandler := NewProxy(1, c)
			handler := testProxyHandler.Create

			requestWrap, _ := createRequestWrapper(tt.requestFiles, strconv.Itoa(tt.distributedPods), string(tt.loadTestType))

			req := httptest.NewRequest("POST", "http://example.com/foo", requestWrap.body)
			req.Header.Set("Content-Type", requestWrap.contentType)
			monkey.Patch(fromHTTPRequestToLoadTestSpec, func(*http.Request, *zap.Logger) (apisLoadTestV1.LoadTestSpec, error) {
				lt := apisLoadTestV1.LoadTestSpec{}
				distributedPods := int32(tt.distributedPods)
				lt.Type = tt.loadTestType
				lt.DistributedPods = &distributedPods
				lt.TestFile = tt.requestFiles["testFile"]
				lt.TestData = tt.requestFiles["testData"]
				lt.EnvVars = tt.requestFiles["envVars"]
				return lt, nil
			})

			w := httptest.NewRecorder()
			handler(w, req)

			resp := w.Result()
			respBody, _ := ioutil.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, resp.Header.Get("Content-Type"), tt.expectedContentType)
			assert.Equal(t, tt.expectedResponse, string(respBody))
		})
	}
}

func TestProxyCreateLimit(t *testing.T) {
	for _, tt := range []struct {
		name             string
		expectedResponse string
		expectedStatus   int
		testsCount       int
		error            error
	}{
		{
			"Limit exceeded",
			"{\"error\":\"Number of active load tests reached limit\"}\n",
			http.StatusTooManyRequests,
			10,
			nil,
		},
		{
			"Can't count tests",
			"{\"error\":\"Could not count active load tests\"}\n",
			http.StatusInternalServerError,
			0,
			errors.New("some error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {})
		logger := zaptest.NewLogger(t)
		ctx := mPkg.SetLogger(context.Background(), logger)

		loadtestClientSet.Fake.PrependReactor("create", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, loadTest, nil
		})
		c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

		testProxyHandler := NewProxy(1, c)
		handler := testProxyHandler.Create

		requestWrap, _ := createRequestWrapper(map[string]string{
			"testFile": "testfile.jmx"}, "2", "Fake")

		req := httptest.NewRequest("POST", "http://example.com/load-test", requestWrap.body)
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", requestWrap.contentType)
		w := httptest.NewRecorder()

		monkey.PatchInstanceMethod(reflect.TypeOf(c), "CountActiveLoadTests", func(*kube.Client, context.Context) (int, error) { return tt.testsCount, tt.error })

		monkey.Patch(fromHTTPRequestToLoadTestSpec, func(*http.Request, *zap.Logger) (apisLoadTestV1.LoadTestSpec, error) {
			lt := apisLoadTestV1.LoadTestSpec{}
			return lt, nil
		})
		handler(w, req)

		resp := w.Result()
		respBody, _ := ioutil.ReadAll(resp.Body)

		assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		assert.Equal(t, tt.expectedResponse, string(respBody))
	}
}

func buildMocFormReq(requestFiles map[string]string, distributedPods, ltType string) (*http.Request, error) {
	request, err := createRequestWrapper(requestFiles, distributedPods, ltType)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "/load-test", request.body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", request.contentType)
	return req, nil
}
