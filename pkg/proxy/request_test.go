package proxy

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestNewFakeFromHTTPLoadTest(t *testing.T) {
	ltType := apisLoadTestV1.LoadTestTypeFake
	r := buildMocFormReq(t, map[string]string{}, "", string(ltType), "", "", "")

	loadTest, err := fromHTTPRequestToLoadTestSpec(r, zaptest.NewLogger(t))
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
			request := buildMocFormReq(t, map[string]string{}, ti.distributedPods, string(apisLoadTestV1.LoadTestTypeJMeter), "", "", "")

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
			request := buildMocFormReq(t, ti.requestFile, "1", string(apisLoadTestV1.LoadTestTypeJMeter), "", "", "")

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
			request := buildMocFormReq(t, ti.requestFile, "1", string(apisLoadTestV1.LoadTestTypeJMeter), "", "", "")

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
		expectedResponse map[string]string
		expectError      bool
	}{
		{
			tag: "valid env vars file",
			requestFile: map[string]string{
				envVars: "testdata/valid/envvars.csv",
			},
			expectedResponse: map[string]string{"envVar1": "value1", "envVar2": "value2"},
			expectError:      false,
		},
		{
			tag: "empty env vars file",
			requestFile: map[string]string{
				envVars: "testdata/invalid/empty.csv",
			},
			expectedResponse: nil,
			expectError:      true,
		},
		{
			tag:              "no env vars file",
			requestFile:      map[string]string{},
			expectedResponse: map[string]string{},
			expectError:      false,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {
			request := buildMocFormReq(t, ti.requestFile, "1", string(apisLoadTestV1.LoadTestTypeJMeter), "", "", "")

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

func TestTags(t *testing.T) {
	testCases := []struct {
		scenario       string
		input          string
		expectedResult apisLoadTestV1.LoadTestTags
		expectedError  string
	}{
		{
			scenario:       "no input no error",
			expectedResult: apisLoadTestV1.LoadTestTags{},
		},
		{
			scenario:       "no value no error",
			input:          ",",
			expectedResult: apisLoadTestV1.LoadTestTags{},
		},
		{
			scenario:      "missing label",
			input:         ":value-only",
			expectedError: "missing tag label",
		},
		{
			scenario:      "missing value",
			input:         "label:",
			expectedError: "missing tag value",
		},
		{
			scenario:      "value is too long",
			input:         "label:MW5Ex91GtG5qTRnC2DIxWo17t6yjkJBCtp9Mh5q0J7R7RXDcoAvRcYmL5Uqc8YeR",
			expectedError: "tag value is too long",
		},
		{
			scenario: "multiple tags",
			input:    "tag1:value1,tag2:value2,,,,tag3:value3",
			expectedResult: apisLoadTestV1.LoadTestTags{
				"tag1": "value1",
				"tag2": "value2",
				"tag3": "value3",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			req := buildMocFormReq(t, nil, "1", string(apisLoadTestV1.LoadTestTypeJMeter), tc.input, "", "")

			result, err := getTags(req)

			assert.Equal(t, tc.expectedResult, result)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
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
		tags            string
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
			tags:            "label:value",
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
			tag: "tag is missing label",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods: "2",
			tags:            ":value",
			expectError:     true,
		},
		{
			tag: "tag is missing value",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods: "2",
			tags:            "label:",
			expectError:     true,
		},
		{
			tag: "tag is too long",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods: "2",
			tags:            "label:MW5Ex91GtG5qTRnC2DIxWo17t6yjkJBCtp9Mh5q0J7R7RXDcoAvRcYmL5Uqc8YeR",
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
			request := buildMocFormReq(t, ti.requestFile, ti.distributedPods, string(ltType), ti.tags, "", "")

			_, err := fromHTTPRequestToLoadTestSpec(request, zaptest.NewLogger(t))

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

	request := buildMocFormReq(t, requestFiles, distributedPods, string(ltType), "label:value", "", "")

	spec, err := fromHTTPRequestToLoadTestSpec(request, zaptest.NewLogger(t))
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

func TestGetImage(t *testing.T) {
	for _, ti := range []struct {
		tag                   string
		role                  string
		imageName             string
		imageTag              string
		expectedImageResponse string
		expectedTagResponse   string
		expectError           bool
	}{
		{
			tag:                   "valid master image",
			role:                  "masterImage",
			imageName:             "hellofresh/kangal-jmeter-master",
			imageTag:              "latest",
			expectedImageResponse: "hellofresh/kangal-jmeter-master",
			expectedTagResponse:   "latest",
			expectError:           false,
		},
		{
			tag:                   "valid worker set",
			role:                  "workerImage",
			imageName:             "hellofresh/kangal-jmeter-worker",
			imageTag:              "latest",
			expectedImageResponse: "hellofresh/kangal-jmeter-worker",
			expectedTagResponse:   "latest",
			expectError:           false,
		},
		{
			tag:                   "valid empty master image and tag",
			role:                  "masterImage",
			imageName:             "",
			imageTag:              "",
			expectedImageResponse: "",
			expectedTagResponse:   "",
			expectError:           false,
		},
		{
			tag:                   "invalid master image name",
			role:                  "masterImage",
			imageName:             "hellofresh/kangal-jmeter-master",
			imageTag:              "latest",
			expectedImageResponse: "hellofresh/kangal-jmeter-worker",
			expectedTagResponse:   "latest",
			expectError:           true,
		},
		{
			tag:                   "invalid worker image tag",
			role:                  "workerImage",
			imageName:             "hellofresh/kangal-jmeter-worker",
			imageTag:              "1.0",
			expectedImageResponse: "hellofresh/kangal-jmeter-worker",
			expectedTagResponse:   "latest",
			expectError:           true,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {

			image := apisLoadTestV1.ImageDetails{
				Image: "",
				Tag:   "",
			}

			sentImage := ti.imageName + ":" + ti.imageTag
			expectedImage := ti.expectedImageResponse + ":" + ti.expectedTagResponse

			if ti.role == "masterImage" {
				request := buildMocFormReq(t, map[string]string{}, "1", string(apisLoadTestV1.LoadTestTypeJMeter), "", sentImage, "")
				image, err = getImage(request, ti.role)
			}
			if ti.role == "workerImage" {
				request := buildMocFormReq(t, map[string]string{}, "1", string(apisLoadTestV1.LoadTestTypeJMeter), "", "", sentImage)
				image, err = getImage(request, ti.role)
			}

			actualImage := image.Image + ":" + image.Tag

			if ti.expectError {
				// assert.Error(t, err)
				assert.NotEqual(t, expectedImage, actualImage)
			} else {
				// assert.NoError(t, err)
				assert.Equal(t, expectedImage, actualImage)
			}

		})
	}
}
