package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"

	"github.com/hellofresh/kangal/pkg/backends"
	_ "github.com/hellofresh/kangal/pkg/backends/jmeter"
	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
	kube "github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	fakeClientset "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
)

const shortDuration = 1 * time.Millisecond // a reasonable duration to block in an example

func TestCreateWithTimeout(t *testing.T) {
	for _, tt := range []struct {
		name             string
		distributedPods  string
		failingLine      string
		requestFiles     map[string]string
		expectedResponse string
	}{
		{
			"Valid JMeter",
			"1",
			"",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			"context deadline exceeded",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := buildMocFormReq(t, tt.requestFiles, tt.distributedPods, "JMeter", "", "", "")

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

func TestProxy_List(t *testing.T) {
	distributedPods := int32(2)
	remainCount := int64(42)

	testCases := []struct {
		scenario            string
		urlParams           string
		result              *apisLoadTestV1.LoadTestList
		error               error
		expectedCode        int
		expectedResponse    string
		expectedContentType string
	}{
		{
			scenario:            "error in client",
			result:              &apisLoadTestV1.LoadTestList{},
			error:               errors.New("client error"),
			expectedCode:        500,
			expectedContentType: "application/json",
			expectedResponse:    `{"error":"client error"}`,
		},
		{
			scenario:            "invalid limit",
			urlParams:           "limit=foobar",
			result:              &apisLoadTestV1.LoadTestList{},
			expectedCode:        400,
			expectedContentType: "application/json",
			expectedResponse:    `{"error":"strconv.ParseInt: parsing \"foobar\": invalid syntax"}`,
		},
		{
			scenario:            "invalid tags",
			urlParams:           "tags=:value",
			result:              &apisLoadTestV1.LoadTestList{},
			expectedCode:        400,
			expectedContentType: "application/json",
			expectedResponse:    `{"error":"missing tag label"}`,
		},
		{
			scenario:            "invalid phase",
			urlParams:           "phase=foo",
			result:              &apisLoadTestV1.LoadTestList{},
			expectedCode:        400,
			expectedContentType: "application/json",
			expectedResponse:    `{"error":"unknown Load Test phase"}`,
		},
		{
			scenario:            "limit is too big",
			urlParams:           "limit=100",
			result:              &apisLoadTestV1.LoadTestList{},
			expectedCode:        400,
			expectedContentType: "application/json",
			expectedResponse:    `{"error":"limit value is too big, max possible value is 50"}`,
		},
		{
			scenario:  "valid phase",
			urlParams: "phase=running",
			result: &apisLoadTestV1.LoadTestList{
				ListMeta: metaV1.ListMeta{
					Continue:           "continue",
					RemainingItemCount: &remainCount,
				},
				Items: []apisLoadTestV1.LoadTest{
					{
						Spec: apisLoadTestV1.LoadTestSpec{
							Type:            apisLoadTestV1.LoadTestTypeJMeter,
							DistributedPods: &distributedPods,
							Tags:            apisLoadTestV1.LoadTestTags{},
							TestFile:        []byte("file content\n"),
							TestData:        []byte("test data\n"),
						},
						Status: apisLoadTestV1.LoadTestStatus{
							Phase:     apisLoadTestV1.LoadTestRunning,
							Namespace: "random",
						},
					},
				},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			expectedResponse:    `{"limit":50,"continue":"continue","remain":42,"items":[{"type":"JMeter","distributedPods":2,"loadtestName":"random","phase":"running","tags":{},"hasEnvVars":false,"hasTestData":true}]}`,
		},
		{
			scenario:  "success",
			urlParams: "tags=department:platform,team:kangal&limit=10",
			result: &apisLoadTestV1.LoadTestList{
				ListMeta: metaV1.ListMeta{
					Continue:           "continue",
					RemainingItemCount: &remainCount,
				},
				Items: []apisLoadTestV1.LoadTest{
					{
						ObjectMeta: metaV1.ObjectMeta{
							Labels: map[string]string{
								"test-tag-department": "platform",
								"test-tag-team":       "kangal",
							},
						},
						Spec: apisLoadTestV1.LoadTestSpec{
							Type:            apisLoadTestV1.LoadTestTypeJMeter,
							Overwrite:       false,
							MasterConfig:    apisLoadTestV1.ImageDetails{},
							WorkerConfig:    apisLoadTestV1.ImageDetails{},
							DistributedPods: &distributedPods,
							Tags:            apisLoadTestV1.LoadTestTags{"department": "platform", "team": "kangal"},
							TestFile:        []byte("file content\n"),
							TestData:        []byte("test data\n"),
						},
						Status: apisLoadTestV1.LoadTestStatus{
							Phase:     apisLoadTestV1.LoadTestRunning,
							Namespace: "random",
						},
					},
				},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			expectedResponse:    `{"limit":10,"continue":"continue","remain":42,"items":[{"type":"JMeter","distributedPods":2,"loadtestName":"random","phase":"running","tags":{"department":"platform","team":"kangal"},"hasEnvVars":false,"hasTestData":true}]}`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			var (
				kubeClientSet     = fake.NewSimpleClientset()
				loadTestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := mPkg.SetLogger(context.Background(), logger)
			loadTestClientSet.Fake.PrependReactor("list", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, tc.result, tc.error
			})
			c := kube.NewClient(loadTestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			testProxyHandler := NewProxy(1, nil, c, 50, false)

			req := httptest.NewRequest("POST", "http://example.com/foo?"+tc.urlParams, nil)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			testProxyHandler.List(w, req)

			resp := w.Result()
			respBody, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tc.expectedCode, resp.StatusCode)
			assert.Equal(t, tc.expectedContentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tc.expectedResponse, strings.Trim(string(respBody), "\n"))
		})
	}
}

func TestProxyCreate(t *testing.T) {
	for _, tt := range []struct {
		name                string
		distributedPods     int
		tagsString          string
		loadTestType        apisLoadTestV1.LoadTestType
		requestFiles        map[string]string
		expectedCode        int
		expectedResponse    string
		expectedContentType string
		creationError       error
	}{
		{
			"Valid request, only test file",
			10,
			"team:kangal",
			apisLoadTestV1.LoadTestTypeJMeter,
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			http.StatusCreated,
			`{"type":"JMeter","distributedPods":10,"phase":"creating","tags":{"team":"kangal"},"hasEnvVars":false,"hasTestData":false}` + "\n",
			"application/json",
			nil,
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
			`{"type":"Fake","distributedPods":10,"phase":"creating","tags":{},"hasEnvVars":true,"hasTestData":true}` + "\n",
			"application/json",
			nil,
		},
		{
			"Invalid loadtest type",
			10,
			"",
			"unknownType",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			http.StatusBadRequest,
			`{"error":"no backend registered for current loadtest type"}` + "\n",
			"application/json",
			errors.New("test creation error"),
		},
		{
			"Error on creation",
			10,
			"",
			apisLoadTestV1.LoadTestTypeFake,
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			http.StatusConflict,
			`{"error":"test creation error"}` + "\n",
			"application/json",
			errors.New("test creation error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				loadTest          = &apisLoadTestV1.LoadTest{}
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := mPkg.SetLogger(context.Background(), logger)
			loadtestClientSet.Fake.PrependReactor("create", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, loadTest, tt.creationError
			})
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			b := backends.New(
				backends.WithLogger(logger),
				backends.WithKubeClientSet(kubeClientSet),
				backends.WithKangalClientSet(loadtestClientSet),
			)

			testProxyHandler := NewProxy(1, b, c, 50, false)
			handler := testProxyHandler.Create

			requestWrap := createRequestWrapper(t, tt.requestFiles, strconv.Itoa(tt.distributedPods), string(tt.loadTestType), tt.tagsString, false, "", "")

			req := httptest.NewRequest("POST", "http://example.com/foo", requestWrap.body)
			req.Header.Set("Content-Type", requestWrap.contentType)

			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler(w, req)

			resp := w.Result()
			respBody, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectedContentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, tt.expectedResponse, string(respBody))
		})
	}
}

func TestNewProxyRecreate(t *testing.T) {
	for _, tt := range []struct {
		name             string
		testsList        *apisLoadTestV1.LoadTestList
		expectedResponse string
		expectedStatus   int
		overwrite        bool
		err              error
	}{
		{
			"Test already exists",
			&apisLoadTestV1.LoadTestList{
				Items: []apisLoadTestV1.LoadTest{
					{
						ObjectMeta: metaV1.ObjectMeta{
							Labels: map[string]string{
								"test-file-hash": "5a7919885ef46f2e0bd66602944128fde2dce928",
							},
						},
						Status: apisLoadTestV1.LoadTestStatus{
							Phase: apisLoadTestV1.LoadTestRunning,
						},
					},
				},
			},
			`{"error":"Load test with given testfile already exists, aborting. Please delete existing load test and try again."}` + "\n",
			http.StatusBadRequest,
			false,
			nil,
		},
		{
			"Can't overwrite existing loadtest",
			&apisLoadTestV1.LoadTestList{
				Items: []apisLoadTestV1.LoadTest{
					{
						ObjectMeta: metaV1.ObjectMeta{
							Labels: map[string]string{
								"test-file-hash": "5a7919885ef46f2e0bd66602944128fde2dce928",
							},
						},
						Status: apisLoadTestV1.LoadTestStatus{
							Phase: apisLoadTestV1.LoadTestRunning,
						},
					},
				},
			},
			`{"error":"loadtests.kangal.hellofresh.com \"\" not found"}` + "\n",
			http.StatusConflict,
			true,
			nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := mPkg.SetLogger(context.Background(), logger)
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)
			b := backends.New(
				backends.WithLogger(logger),
				backends.WithKubeClientSet(kubeClientSet),
				backends.WithKangalClientSet(loadtestClientSet),
			)

			loadtestClientSet.Fake.PrependReactor("list", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, tt.testsList, tt.err
			})

			requestFiles := map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			}
			requestWrap := createRequestWrapper(t, requestFiles, "2", "Fake", "", tt.overwrite, "", "")

			req := httptest.NewRequest("POST", "http://example.com/load-test", requestWrap.body)
			req = req.WithContext(ctx)
			req.Header.Set("Content-Type", requestWrap.contentType)

			w := httptest.NewRecorder()

			testProxyHandler := NewProxy(1, b, c, 50, false)
			testProxyHandler.Create(w, req)

			resp := w.Result()
			respBody, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
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
			`{"error":"Number of active load tests reached limit"}` + "\n",
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
			`{"error":"Could not count active load tests"}` + "\n",
			http.StatusInternalServerError,
			&apisLoadTestV1.LoadTestList{
				Items: []apisLoadTestV1.LoadTest{},
			},
			errors.New("some error"),
			nil,
		},
		{
			"Can't count labeled tests",
			`{"error":"Could not count active load tests with given hash"}` + "\n",
			http.StatusInternalServerError,
			&apisLoadTestV1.LoadTestList{},
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
			loadtestClientSet.Fake.PrependReactor("list", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
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
			b := backends.New(
				backends.WithLogger(logger),
				backends.WithKubeClientSet(kubeClientSet),
				backends.WithKangalClientSet(loadtestClientSet),
			)

			requestFiles := map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			}
			requestWrap := createRequestWrapper(t, requestFiles, "2", "Fake", "", false, "", "")

			req := httptest.NewRequest("POST", "http://example.com/load-test", requestWrap.body)
			req = req.WithContext(ctx)
			req.Header.Set("Content-Type", requestWrap.contentType)
			w := httptest.NewRecorder()

			testProxyHandler := NewProxy(1, b, c, 50, false)
			testProxyHandler.Create(w, req)

			resp := w.Result()
			respBody, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedResponse, string(respBody))
		})
	}
}

func TestProxyGet(t *testing.T) {
	var pods = int32(1)
	for _, tt := range []struct {
		name             string
		loadTest         apisLoadTestV1.LoadTest
		expectedCode     int
		expectedResponse string
		error            error
	}{
		{
			"Valid request",
			apisLoadTestV1.LoadTest{
				Spec: apisLoadTestV1.LoadTestSpec{
					Type:            apisLoadTestV1.LoadTestTypeJMeter,
					DistributedPods: &pods,
					Tags: apisLoadTestV1.LoadTestTags{
						"team": "kangal",
					},
				},
				Status: apisLoadTestV1.LoadTestStatus{
					Phase:     apisLoadTestV1.LoadTestRunning,
					Namespace: "aaa",
				}},
			http.StatusOK,
			`{"type":"JMeter","distributedPods":1,"loadtestName":"aaa","phase":"running","tags":{"team":"kangal"},"hasEnvVars":false,"hasTestData":false}` + "\n",
			nil,
		},
		{
			"Error",
			apisLoadTestV1.LoadTest{},
			http.StatusInternalServerError,
			`{"error":"some test error"}` + "\n",
			errors.New("some test error"),
		},
		{
			"Not found",
			apisLoadTestV1.LoadTest{},
			http.StatusNotFound,
			`{"error":"loadtest.kangal.hellofresh.com \"name\" not found"}` + "\n",
			k8sAPIErrors.NewNotFound(apisLoadTestV1.Resource("loadtest"), "name"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := mPkg.SetLogger(context.Background(), logger)
			loadtestClientSet.Fake.PrependReactor("get", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &tt.loadTest, tt.error
			})
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)
			b := backends.New(
				backends.WithLogger(logger),
			)

			req := httptest.NewRequest("GET", "http://example.com/load-test/testname", nil)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			testProxyHandler := NewProxy(1, b, c, 50, false)
			testProxyHandler.Get(w, req)

			resp := w.Result()
			respBody, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectedResponse, string(respBody))
		})
	}
}

func TestProxyDelete(t *testing.T) {
	for _, tt := range []struct {
		name             string
		expectedCode     int
		expectedResponse string
		error            error
	}{
		{
			"Delete test",
			http.StatusNoContent,
			"",
			nil,
		},
		{
			"Error on deleting test",
			http.StatusBadRequest,
			`{"error":"some error"}` + "\n",
			errors.New("some error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				loadTest          = &apisLoadTestV1.LoadTest{}
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := mPkg.SetLogger(context.Background(), logger)
			loadtestClientSet.Fake.PrependReactor("delete", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, loadTest, tt.error
			})
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			req := httptest.NewRequest("DELETE", "http://example.com/load-test/some-test", nil)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			testProxyHandler := NewProxy(1, nil, c, 50, false)
			testProxyHandler.Delete(w, req)

			resp := w.Result()
			respBody, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectedResponse, string(respBody))
		})
	}
}

func TestProxyGetLogs(t *testing.T) {
	var (
		pods = int32(1)
	)
	for _, tt := range []struct {
		name             string
		loadTest         apisLoadTestV1.LoadTest
		expectedCode     int
		expectedResponse string
		ltError          error
		podError         error
		ltID             string
	}{
		{
			"No content",
			apisLoadTestV1.LoadTest{
				Status: apisLoadTestV1.LoadTestStatus{
					Phase:     apisLoadTestV1.LoadTestRunning,
					Namespace: "",
				}},
			http.StatusNoContent,
			`{"error":"no logs found in load test resources"}` + "\n",
			nil,
			nil,
			"",
		},
		{
			"Error on getting master pod",
			apisLoadTestV1.LoadTest{
				Status: apisLoadTestV1.LoadTestStatus{
					Phase:     apisLoadTestV1.LoadTestRunning,
					Namespace: "aaa",
				}},
			http.StatusBadRequest,
			`{"error":"error on listing pods"}` + "\n",
			nil,
			errors.New("error on listing pods"),
			"",
		},
		{
			"Can't get load test",
			apisLoadTestV1.LoadTest{
				Spec: apisLoadTestV1.LoadTestSpec{
					Type:            apisLoadTestV1.LoadTestTypeJMeter,
					Overwrite:       false,
					DistributedPods: &pods,
				},
				Status: apisLoadTestV1.LoadTestStatus{
					Phase:     apisLoadTestV1.LoadTestRunning,
					Namespace: "aaa",
				},
			},
			http.StatusBadRequest,
			`{"error":"error on getting loadtest"}` + "\n",
			errors.New("error on getting loadtest"),
			nil,
			"",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := mPkg.SetLogger(context.Background(), logger)
			loadtestClientSet.Fake.PrependReactor("get", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &tt.loadTest, tt.ltError
			})
			kubeClientSet.Fake.PrependReactor("list", "pods", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &corev1.PodList{}, tt.podError
			})
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)
			b := backends.New(
				backends.WithLogger(logger),
				backends.WithKubeClientSet(kubeClientSet),
				backends.WithKangalClientSet(loadtestClientSet),
			)

			routeCtx := new(chi.Context)
			routeCtx.URLParams.Add(loadTestID, tt.ltID)

			ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)

			req := httptest.NewRequest("GET", "http://example.com/load-test/some-test/logs", nil)

			w := httptest.NewRecorder()

			testProxyHandler := NewProxy(1, b, c, 50, false)
			testProxyHandler.GetLogs(w, req.WithContext(ctx))

			resp := w.Result()
			respBody, _ := io.ReadAll(resp.Body)

			require.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectedResponse, string(respBody))
		})
	}

}

func buildMocFormReq(t *testing.T, requestFiles map[string]string, distributedPods, ltType, tagsString string, masterImage string, workerImage string) *http.Request {
	t.Helper()

	request := createRequestWrapper(t, requestFiles, distributedPods, ltType, tagsString, false, masterImage, workerImage)

	req, err := http.NewRequest("POST", "/load-test", request.body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", request.contentType)
	return req
}
