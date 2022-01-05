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
	r := buildMocFormReq(t, map[string]string{testFile: "testdata/valid/loadtest.jmx"}, "", string(ltType), "", "", "")

	loadTest, err := fromHTTPRequestToLoadTestSpec(r, zaptest.NewLogger(t), false)
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
			request := buildMocFormReq(t, map[string]string{testFile: "testdata/valid/loadtest.jmx"}, ti.distributedPods, string(apisLoadTestV1.LoadTestTypeJMeter), "", "", "")

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
			tag:              "error when no test file",
			requestFile:      map[string]string{},
			expectedResponse: "",
			expectError:      true,
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
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			expectedResponse: "test data 1\ntest data 2\n",
			expectError:      false,
		},
		{
			tag: "empty test data file",
			requestFile: map[string]string{
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/invalid/empty.csv",
			},
			expectedResponse: "",
			expectError:      true,
		},
		{
			tag: "no test data file specified",
			requestFile: map[string]string{
				testFile: "testdata/valid/loadtest.jmx",
			},
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
				testFile: "testdata/valid/loadtest.jmx",
				envVars:  "testdata/valid/envvars.csv",
			},
			expectedResponse: map[string]string{"envVar1": "value1", "envVar2": "value2"},
			expectError:      false,
		},
		{
			tag: "invalid env vars file format",
			requestFile: map[string]string{
				testFile: "testdata/valid/loadtest.jmx",
				envVars:  "testdata/valid/loadtest.jmx",
			},
			expectedResponse: nil,
			expectError:      true,
		},
		{
			tag: "empty env vars file",
			requestFile: map[string]string{
				testFile: "testdata/valid/loadtest.jmx",
				envVars:  "testdata/invalid/empty.csv",
			},
			expectedResponse: nil,
			expectError:      true,
		},
		{
			tag: "no env vars file",
			requestFile: map[string]string{
				testFile: "testdata/valid/loadtest.jmx",
			},
			expectedResponse: map[string]string(nil),
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

			req := buildMocFormReq(t, map[string]string{testFile: "testdata/valid/loadtest.jmx"}, "1", string(apisLoadTestV1.LoadTestTypeJMeter), tc.input, "", "")

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

			_, err := fromHTTPRequestToLoadTestSpec(request, zaptest.NewLogger(t), false)

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

	spec, err := fromHTTPRequestToLoadTestSpec(request, zaptest.NewLogger(t), false)
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

func TestGetTargetURL(t *testing.T) {
	for _, ti := range []struct {
		tag         string
		targetURL   string
		expected    string
		expectError bool
	}{
		{
			tag:         "valid URL as targetURL",
			targetURL:   "https://test-url.com/foo",
			expected:    "https://test-url.com/foo",
			expectError: false,
		},
		{
			tag:         "invalid URL without scheme",
			targetURL:   "someurls.com/foo-test",
			expected:    "",
			expectError: true,
		},
		{
			tag:         "invalid URL without host",
			targetURL:   "http://",
			expected:    "",
			expectError: true,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/load-test", new(bytes.Buffer))
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			req.Form = url.Values{"targetURL": []string{ti.targetURL}}
			req.ParseForm()

			actual, err := getTargetURL(req)

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, ti.expected, actual)

		})
	}
}

func TestGetImage(t *testing.T) {
	for _, ti := range []struct {
		tag              string
		role             string
		imageName        string
		imageTag         string
		expectedResponse string
		expectError      bool
	}{
		{
			tag:              "valid master image",
			role:             "masterImage",
			imageName:        "hellofresh/kangal-jmeter-master",
			imageTag:         "latest",
			expectedResponse: "hellofresh/kangal-jmeter-master:latest",
			expectError:      false,
		},
		{
			tag:              "valid worker set",
			role:             "workerImage",
			imageName:        "hellofresh/kangal-jmeter-worker",
			imageTag:         "latest",
			expectedResponse: "hellofresh/kangal-jmeter-worker:latest",
			expectError:      false,
		},
		{
			tag:              "valid empty master image and tag",
			role:             "masterImage",
			imageName:        "",
			imageTag:         "",
			expectedResponse: "",
			expectError:      false,
		},
		{
			tag:              "invalid master image name",
			role:             "masterImage",
			imageName:        "hellofresh/kangal-jmeter-master",
			imageTag:         "latest",
			expectedResponse: "hellofresh/kangal-jmeter-worker:latest",
			expectError:      true,
		},
		{
			tag:              "invalid worker image tag",
			role:             "workerImage",
			imageName:        "hellofresh/kangal-jmeter-worker",
			imageTag:         "1.0",
			expectedResponse: "hellofresh/kangal-jmeter-worker:latest",
			expectError:      true,
		},
		{
			tag:              "image without tag",
			role:             "workerImage",
			imageName:        "hellofresh/kangal-jmeter-worker",
			imageTag:         "",
			expectedResponse: "",
			expectError:      false,
		},
		{
			tag:              "host registry includes port",
			role:             "workerImage",
			imageName:        "test.com:5000/hellofresh/hellofreshkangal-jmeter-worker",
			imageTag:         "v1.7",
			expectedResponse: "test.com:5000/hellofresh/hellofreshkangal-jmeter-worker:v1.7",
			expectError:      false,
		},
		{
			tag:              "Empty tag",
			role:             "workerImage",
			imageName:        "test.com:5000/hellofresh/hellofreshkangal-jmeter-worker",
			imageTag:         "",
			expectedResponse: "",
			expectError:      false,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {

			image := apisLoadTestV1.ImageDetails{
				Image: "",
				Tag:   "",
			}

			sentImage := ""
			if (ti.imageName != "") && (ti.imageTag != "") {
				sentImage = ti.imageName + ":" + ti.imageTag
			}

			if ti.role == "masterImage" {
				request := buildMocFormReq(t, map[string]string{testFile: "testdata/valid/loadtest.jmx"}, "1", string(apisLoadTestV1.LoadTestTypeJMeter), "", sentImage, "")
				image = getImage(request, ti.role)
			}
			if ti.role == "workerImage" {
				request := buildMocFormReq(t, map[string]string{testFile: "testdata/valid/loadtest.jmx"}, "1", string(apisLoadTestV1.LoadTestTypeJMeter), "", "", sentImage)
				image = getImage(request, ti.role)
			}

			actualImage := ""
			if (image.Image != "") && (image.Tag != "") {
				actualImage = image.Image + ":" + image.Tag
			}

			if ti.expectError {
				assert.NotEqual(t, ti.expectedResponse, actualImage)
			} else {
				assert.Equal(t, ti.expectedResponse, actualImage)
			}

		})
	}
}

func TestCustomImageFeatureFlag(t *testing.T) {
	for _, ti := range []struct {
		tag                     string
		allowedCustomImages     bool
		masterImage             string
		workerImage             string
		expectedMasterImageName string
		expectedMasterImageTag  string
		expectedWorkerImageName string
		expectedWorkerImageTag  string
		expectError             bool
	}{
		{
			tag:                     "Allowed custom images, defined custom images",
			allowedCustomImages:     true,
			masterImage:             "fake/masterImage:v1",
			workerImage:             "fake/workerImage:v1",
			expectedMasterImageName: "fake/masterImage",
			expectedMasterImageTag:  "v1",
			expectedWorkerImageName: "fake/workerImage",
			expectedWorkerImageTag:  "v1",
			expectError:             false,
		},
		{
			tag:                     "Disallowed custom images, defined custom images",
			allowedCustomImages:     false,
			masterImage:             "fake/masterImage:v1",
			workerImage:             "fake/workerImage:v1",
			expectedMasterImageName: "",
			expectedMasterImageTag:  "",
			expectedWorkerImageName: "",
			expectedWorkerImageTag:  "",
			expectError:             false,
		},
		{
			tag:                     "Allowed custom images, undefined custom images",
			allowedCustomImages:     true,
			masterImage:             "",
			workerImage:             "",
			expectedMasterImageName: "",
			expectedMasterImageTag:  "",
			expectedWorkerImageName: "",
			expectedWorkerImageTag:  "",
			expectError:             false,
		},
		{
			tag:                     "Wrong Image format",
			allowedCustomImages:     true,
			masterImage:             "this/is/not/a/correct/image:format",
			workerImage:             "this/is/not/a/correct/image:format",
			expectedMasterImageName: "",
			expectedMasterImageTag:  "",
			expectedWorkerImageName: "",
			expectedWorkerImageTag:  "",
			expectError:             false,
		},
		{
			tag:                     "Allowed custom, only master defined",
			allowedCustomImages:     true,
			masterImage:             "fake/masterImage:v1",
			workerImage:             "",
			expectedMasterImageName: "fake/masterImage",
			expectedMasterImageTag:  "v1",
			expectedWorkerImageName: "",
			expectedWorkerImageTag:  "",
			expectError:             false,
		},
	} {

		t.Run(ti.tag, func(t *testing.T) {
			request := buildMocFormReq(t, map[string]string{testFile: "testdata/valid/loadtest.jmx"}, "1", string(apisLoadTestV1.LoadTestTypeJMeter), "", ti.masterImage, ti.workerImage)

			ltSpec, err := fromHTTPRequestToLoadTestSpec(request, zaptest.NewLogger(t), ti.allowedCustomImages)

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if ti.expectError {
				assert.NotEqual(t, ti.expectedMasterImageName, ltSpec.MasterConfig.Image)
				assert.NotEqual(t, ti.expectedMasterImageTag, ltSpec.MasterConfig.Tag)
				assert.NotEqual(t, ti.expectedWorkerImageName, ltSpec.WorkerConfig.Image)
				assert.NotEqual(t, ti.expectedWorkerImageTag, ltSpec.WorkerConfig.Tag)
			} else {
				assert.Equal(t, ti.expectedMasterImageName, ltSpec.MasterConfig.Image)
				assert.Equal(t, ti.expectedMasterImageTag, ltSpec.MasterConfig.Tag)
				assert.Equal(t, ti.expectedWorkerImageName, ltSpec.WorkerConfig.Image)
				assert.Equal(t, ti.expectedWorkerImageTag, ltSpec.WorkerConfig.Tag)
			}

		})

	}
}

func Test_getTypeFromName(t *testing.T) {
	for _, ti := range []struct {
		tag          string
		filename     string
		expectedType string
	}{
		{
			tag:          "no filename provided",
			filename:     "",
			expectedType: "",
		},
		{
			tag:          "filename with no extension",
			filename:     "somefile",
			expectedType: "",
		},
		{
			tag:          "filename with extension",
			filename:     "somefile.ext",
			expectedType: "ext",
		},
		{
			tag:          "filename with multiple extensions",
			filename:     "somefile.tar.gz",
			expectedType: "gz",
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {
			ext := getTypeFromName(ti.filename)
			assert.Equal(t, ti.expectedType, ext)
		})
	}
}
