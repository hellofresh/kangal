package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestNewJMeterFromHTTPLoadTest(t *testing.T) {
	r, err := buildMocFormReq(map[string]string{}, "", string(apisLoadTestV1.LoadTestTypeJMeter))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	loadTest, err := FromHTTPRequestToJMeter(r, zap.NewNop())
	require.Error(t, err)
	assert.Nil(t, loadTest)
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
	expectedDP := int32(2)
	for _, ti := range []struct {
		tag              string
		requestFile      map[string]string
		distributedPods  string
		expectedResponse *apisLoadTestV1.LoadTestSpec
		expectError      bool
	}{
		{
			tag: "valid request",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods: "2",
			expectedResponse: &apisLoadTestV1.LoadTestSpec{
				Type:            apisLoadTestV1.LoadTestTypeJMeter,
				DistributedPods: &expectedDP,
				TestFile:        "load-test file\n",
				TestData:        "test data 1\ntest data 2\n",
				EnvVars:         "envVar1,value1\nenvVar2,value2\n",
			},
			expectError: false,
		},
		{
			tag: "distributed pods is invalid",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods:  "aa",
			expectedResponse: nil,
			expectError:      true,
		},
		{
			tag: "empty envvars file",
			requestFile: map[string]string{
				envVars:  "testdata/invalid/empty.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods:  "2",
			expectedResponse: nil,
			expectError:      true,
		},
		{
			tag: "empty test file",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/invalid/empty.jmx",
				testData: "testdata/valid/testdata.csv",
			},
			distributedPods:  "2",
			expectedResponse: nil,
			expectError:      true,
		},
		{
			tag: "empty test data",
			requestFile: map[string]string{
				envVars:  "testdata/valid/envvars.csv",
				testFile: "testdata/valid/loadtest.jmx",
				testData: "testdata/invalid/empty.csv",
			},
			distributedPods:  "2",
			expectedResponse: nil,
			expectError:      true,
		},
	} {

		t.Run(ti.tag, func(t *testing.T) {
			request, err := buildMocFormReq(ti.requestFile, ti.distributedPods, string(apisLoadTestV1.LoadTestTypeJMeter))
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			spec, err := FromHTTPRequestToJMeter(request, zap.NewNop())
			assert.Equal(t, ti.expectedResponse, spec)

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})

	}
}

func TestHash(t *testing.T) {
	jmeter := &JMeter{
		Spec: &apisLoadTestV1.LoadTestSpec{
			Type:     apisLoadTestV1.LoadTestTypeJMeter,
			TestFile: "",
		},
	}

	assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", jmeter.Hash())
}

func TestJMeterCR(t *testing.T) {
	expectedDP := int32(2)
	requestFiles := map[string]string{
		envVars:  "testdata/valid/envvars.csv",
		testFile: "testdata/valid/loadtest.jmx",
		testData: "testdata/valid/testdata.csv",
	}
	distributedPods := "2"
	jmeter := &JMeter{
		Spec: &apisLoadTestV1.LoadTestSpec{
			Type:            apisLoadTestV1.LoadTestTypeJMeter,
			DistributedPods: &expectedDP,
			TestFile:        "load-test file\n",
			TestData:        "test data 1\ntest data 2\n",
			EnvVars:         "envVar1,value1\nenvVar2,value2\n",
		},
	}

	request, err := buildMocFormReq(requestFiles, distributedPods, string(apisLoadTestV1.LoadTestTypeJMeter))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	spec, err := FromHTTPRequestToJMeter(request, zap.NewNop())
	require.NoError(t, err)

	jm, err := NewJMeterLoadTest(spec, zap.NewNop())

	assert.NoError(t, err)
	assert.Equal(t, jmeter.Spec, jm.Spec)
}

func TestRequestValidatorValidSpec(t *testing.T) {
	var distributedPods int32 = 3

	loadTest := &JMeter{
		Spec: &apisLoadTestV1.LoadTestSpec{
			Type:            apisLoadTestV1.LoadTestTypeJMeter,
			TestFile:        "asdf",
			DistributedPods: &distributedPods,
		},
		Logger: zap.NewNop(),
	}

	err := loadTest.validate()
	assert.NoError(t, err)
}

func TestRequestValidatorType(t *testing.T) {
	loadTest := &JMeter{
		Spec:   &apisLoadTestV1.LoadTestSpec{},
		Logger: zap.NewNop(),
	}

	err := loadTest.validate()
	assert.Equal(t, ErrRequiredJMeterType, err)
}

func TestRequestValidatorHasSpec(t *testing.T) {
	loadTest := &JMeter{
		Spec:   nil,
		Logger: zap.NewNop(),
	}

	err := loadTest.validate()
	assert.Equal(t, ErrEmptySpec, err)
}

func TestRequestValidatorNoPods(t *testing.T) {
	loadTest := &JMeter{
		Spec: &apisLoadTestV1.LoadTestSpec{
			Type:            apisLoadTestV1.LoadTestTypeJMeter,
			TestFile:        "asdf",
			DistributedPods: nil,
		},
		Logger: zap.NewNop(),
	}

	err := loadTest.validate()
	assert.Equal(t, ErrRequireMinOneDistributedPod, err)
}

func TestRequestValidatorWrongPods(t *testing.T) {
	var distributedPods int32 = 0

	loadTest := &JMeter{
		Spec: &apisLoadTestV1.LoadTestSpec{
			Type:            apisLoadTestV1.LoadTestTypeJMeter,
			TestFile:        "asdf",
			DistributedPods: &distributedPods,
		},
		Logger: zap.NewNop(),
	}

	err := loadTest.validate()
	assert.Equal(t, ErrRequireMinOneDistributedPod, err)
}

func TestRequestValidatorNoTestFile(t *testing.T) {
	var distributedPods int32 = 2

	loadTest := &JMeter{
		Spec: &apisLoadTestV1.LoadTestSpec{
			Type:            apisLoadTestV1.LoadTestTypeJMeter,
			TestFile:        "",
			DistributedPods: &distributedPods,
		},
		Logger: zap.NewNop(),
	}

	err := loadTest.validate()
	assert.Equal(t, ErrRequireTestFile, err)
}
