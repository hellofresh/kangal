package proxy

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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
			var (
				loadTest          = &apisLoadTestV1.LoadTest{}
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)

			loadtestClientSet.Fake.PrependReactor("create", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, loadTest, nil
			})
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			s := specData{
				distributedPods: tt.distributedPods,
				ltType:          string(tt.loadTestType),
				files:           tt.requestFiles,
				err:             nil,
			}

			testProxyHandler := NewProxy(1, c, s.fakeSpecCreator)
			handler := testProxyHandler.Create

			requestWrap, _ := createRequestWrapper(tt.requestFiles, strconv.Itoa(tt.distributedPods), string(tt.loadTestType))

			req := httptest.NewRequest("POST", "http://example.com/foo", requestWrap.body)
			req.Header.Set("Content-Type", requestWrap.contentType)

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

func TestProxyCreateWithErrors(t *testing.T) {
	for _, tt := range []struct {
		name             string
		expectedResponse string
		expectedStatus   int
		testsList        *apisLoadTestV1.LoadTestList
		error            error
		listLabeledError error
	}{
		{
			"Limit exceeded",
			"{\"error\":\"Number of active load tests reached limit\"}\n",
			http.StatusTooManyRequests,
			&apisLoadTestV1.LoadTestList{
				Items: []apisLoadTestV1.LoadTest{
					{
						Status: apisLoadTestV1.LoadTestStatus{
							Phase: apisLoadTestV1.LoadTestRunning,
						},
					},
				},
			},
			nil,
			nil,
		},
		{
			"Can't count tests",
			"{\"error\":\"Could not count active load tests\"}\n",
			http.StatusInternalServerError,
			&apisLoadTestV1.LoadTestList{
				Items: []apisLoadTestV1.LoadTest{},
			},
			errors.New("some error"),
			nil,
		},
		{
			"Can't count labeled tests",
			"{\"error\":\"Could not count active load tests with given hash\"}\n",
			http.StatusInternalServerError,
			&apisLoadTestV1.LoadTestList{
				Items: []apisLoadTestV1.LoadTest{},
			},
			nil,
			errors.New("some error on counting labeled tests"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := mPkg.SetLogger(context.Background(), logger)

			var listCalls int

			// we should use PrependReactor to add a new mock in the beginning of the Action list
			// because by default ReactionChain has '*'/'*' in the beginning of new list
			loadtestClientSet.Fake.PrependReactor("list", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				listCalls++
				// We have 2 calls of (c *loadTests) List in create method.
				// The first in GetLoadTestsByLabel, the second in CountActiveLoadTests.
				// For this test we want to skip the first call and always return an empty list.
				switch listCalls {
				case 1:
					return true, &apisLoadTestV1.LoadTestList{}, tt.listLabeledError
				case 2:
					return true, tt.testsList, tt.error
				default:
					return true, nil, errors.New("unexpected number of calls")
				}
			})

			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			s := specData{}
			testProxyHandler := NewProxy(1, c, s.fakeSpecCreator)
			handler := testProxyHandler.Create

			requestWrap, _ := createRequestWrapper(map[string]string{
				"testFile": "testfile.jmx"}, "2", "Fake")

			req := httptest.NewRequest("POST", "http://example.com/load-test", requestWrap.body)
			req = req.WithContext(ctx)
			req.Header.Set("Content-Type", requestWrap.contentType)
			w := httptest.NewRecorder()

			handler(w, req)

			resp := w.Result()
			respBody, _ := ioutil.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedResponse, string(respBody))
		})
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

type specData struct {
	distributedPods int
	ltType          string
	files           map[string]string
	err             error
}

func (s *specData) fakeSpecCreator(*http.Request, *zap.Logger) (apisLoadTestV1.LoadTestSpec, error) {
	lt := apisLoadTestV1.LoadTestSpec{}
	distributedPods := int32(s.distributedPods)
	lt.Type = apisLoadTestV1.LoadTestType(s.ltType)
	lt.DistributedPods = &distributedPods
	lt.TestFile = s.files["testFile"]
	lt.TestData = s.files["testData"]
	lt.EnvVars = s.files["envVars"]
	return lt, s.err
}
