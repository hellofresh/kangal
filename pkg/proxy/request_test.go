package proxy

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestNewFakeFromHTTPLoadTest(t *testing.T) {
	ltType := apisLoadTestV1.LoadTestTypeFake
	r, err := buildMocFormReq(map[string]string{}, "", string(ltType))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	loadTest, err := fromHTTPRequestToLoadTestSpec(r, zap.NewNop())
	require.Error(t, err)
	assert.Equal(t, apisLoadTestV1.LoadTestSpec{}, loadTest)
}

func TestDistributedPods(t *testing.T) {
	for _, ti := range []struct {
		tag              string
		distributedPods  string
		expectedResponse int32
		expectError      bool
	}{
		{
			tag:              "valid distributed pods value",
			distributedPods:  "2",
			expectedResponse: 2,
			expectError:      false,
		},
		{
			tag:              "invalid char distributed pods value",
			distributedPods:  "a",
			expectedResponse: 0,
			expectError:      true,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {
			request, err := buildMocFormReq(map[string]string{}, ti.distributedPods, string(apisLoadTestV1.LoadTestTypeJMeter))
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			n, err := getDistributedPods(request)
			assert.Equal(t, n, ti.expectedResponse)

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTestFile(t *testing.T) {
	for _, ti := range []struct {
		tag              string
		requestFile      map[string]string
		expectedResponse string
		expectError      bool
	}{
		{
			tag: "valid test file",
			requestFile: map[string]string{
				testFile: "testdata/valid/loadtest.jmx",
			},
			expectedResponse: "load-test file\n",
			expectError:      false,
		},
		{
			tag: "invalid empty test file",
			requestFile: map[string]string{
				testFile: "testdata/invalid/empty.jmx",
			},
			expectedResponse: "",
			expectError:      true,
		},
		{
			tag:              "valid no test file",
			requestFile:      map[string]string{},
			expectedResponse: "",
			expectError:      false,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {
			request, err := buildMocFormReq(ti.requestFile, "1", string(apisLoadTestV1.LoadTestTypeJMeter))
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			n, err := getTestFile(request)
			assert.Equal(t, ti.expectedResponse, n)

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDataFile(t *testing.T) {
	for _, ti := range []struct {
		tag              string
		requestFile      map[string]string
		expectedResponse string
		expectError      bool
	}{
		{
			tag: "valid test data",
			requestFile: map[string]string{
				testData: "testdata/valid/testdata.csv",
			},
			expectedResponse: "test data 1\ntest data 2\n",
			expectError:      false,
		},
		{
			tag: "empty test data file",
			requestFile: map[string]string{
				testData: "testdata/invalid/empty.csv",
			},
			expectedResponse: "",
			expectError:      true,
		},
		{
			tag:              "no test data file specified",
			requestFile:      map[string]string{},
			expectedResponse: "",
			expectError:      false,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {
			request, err := buildMocFormReq(ti.requestFile, "1", string(apisLoadTestV1.LoadTestTypeJMeter))
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			n, err := getTestData(request)
			assert.Equal(t, ti.expectedResponse, n)

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnvVarFile(t *testing.T) {
	for _, ti := range []struct {
		tag              string
		requestFile      map[string]string
		expectedResponse string
		expectError      bool
	}{
		{
			tag: "valid env vars file",
			requestFile: map[string]string{
				envVars: "testdata/valid/envvars.csv",
			},
			expectedResponse: "envVar1,value1\nenvVar2,value2\n",
			expectError:      false,
		},
		{
			tag: "empty env vars file",
			requestFile: map[string]string{
				envVars: "testdata/invalid/empty.csv",
			},
			expectedResponse: "",
			expectError:      true,
		},
		{
			tag:              "no env vars file",
			requestFile:      map[string]string{},
			expectedResponse: "",
			expectError:      false,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {
			request, err := buildMocFormReq(ti.requestFile, "1", string(apisLoadTestV1.LoadTestTypeJMeter))
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			n, err := getEnvVars(request)
			assert.Equal(t, ti.expectedResponse, n)

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInit(t *testing.T) {
	ltType := apisLoadTestV1.LoadTestTypeJMeter
	for _, ti := range []struct {
		tag             string
		requestFile     map[string]string
		distributedPods string
		expectError     bool
	}{
		{
			tag: "valid request",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods: "2",
			expectError:     false,
		},
		{
			tag: "invalid request - wrong testFile format",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/envvars.csv",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods: "2",
			expectError:     true,
		},
		{
			tag: "invalid request - wrong testData format",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/loadtest.jmx",
			},
			distributedPods: "2",
			expectError:     true,
		},
		{
			tag: "invalid request - wrong envVars format",
			requestFile: map[string]string{
				envVars:  "testdata/valid/loadtest.jmx",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/envvars.csv",
			},
			distributedPods: "2",
			expectError:     true,
		},
		{
			tag: "distributed pods is invalid",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods: "aa",
			expectError:     true,
		},
	} {

		t.Run(ti.tag, func(t *testing.T) {
			request, err := buildMocFormReq(ti.requestFile, ti.distributedPods, string(ltType))
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			_, err = fromHTTPRequestToLoadTestSpec(request, zap.NewNop())

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})

	}
}

func TestCheckLoadTestSpec(t *testing.T) {
	ltType := apisLoadTestV1.LoadTestTypeJMeter
	requestFiles := map[string]string{
		envVars:  "testdata/valid/envvars.csv",
		testFile: "testdata/valid/loadtest.jmx",
		testData: "testdata/valid/testdata.csv",
	}
	distributedPods := "2"

	request, err := buildMocFormReq(requestFiles, distributedPods, string(ltType))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	spec, err := fromHTTPRequestToLoadTestSpec(request, zap.NewNop())
	require.NoError(t, err)

	lt, err := apisLoadTestV1.BuildLoadTestObject(spec)
	assert.NoError(t, err)
	assert.NotEmpty(t, lt.Name)
	assert.NotEmpty(t, lt.Labels)
}

func TestGetDuration(t *testing.T) {
	scenarios := []struct {
		duration    string
		expected    time.Duration
		expectError bool
	}{
		{
			duration:    "1m",
			expected:    1 * time.Minute,
			expectError: false,
		},
		{
			duration:    "1d",
			expected:    time.Duration(0),
			expectError: true,
		},
		{
			duration:    "",
			expected:    time.Duration(0),
			expectError: false,
		},
	}

	for _, scenario := range scenarios {
		req, err := http.NewRequest("POST", "/load-test", new(bytes.Buffer))
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		req.Form = url.Values{"duration": []string{scenario.duration}}
		req.ParseForm()

		actual, err := getDuration(req)

		if scenario.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, scenario.expected, actual)
	}
}
