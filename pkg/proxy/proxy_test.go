package proxy

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConvertTestName(t *testing.T) {
	testName := "ederE_rfrg.jmx"
	convertedName := convertTestName(testName)

	assert.Equal(t, "edere-rfrg", convertedName)
}

func TestConvertTestNameWithoutSuffix(t *testing.T) {
	testName := "123_TEST_FILE"
	convertedName := convertTestName(testName)

	assert.Equal(t, "123-test-file", convertedName)
}

func TestConvertTestNameSpecialSymbols(t *testing.T) {
	testName := "¨¨ƒ¸¸dsgc_ŕtdv"
	convertedName := convertTestName(testName)

	assert.Equal(t, "¨¨ƒ¸¸dsgc-ŕtdv", convertedName)
}

func TestRequestValidator(t *testing.T) {
	requestFiles := map[string]string{
		"testFile": "testdata/valid/loadtest.jmx",
	}

	distributedPods := "2"

	request, err := buildMocFormReq(requestFiles, distributedPods, "JMeter")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	spec, err := FromHTTPRequestToJMeter(request, "JMeter", zap.NewNop())
	require.NoError(t, err)

	loadTest := &JMeter{
		Spec:   spec,
		Logger: zap.NewNop(),
	}

	err = loadTest.validate()
	assert.NoError(t, err)
}

func TestRequestValidatorWrongTestFile(t *testing.T) {
	requestFiles := map[string]string{
		"testFile": "testdata/valid/testdata.csv",
	}

	distributedPods := "2"

	request, err := buildMocFormReq(requestFiles, distributedPods, "JMeter")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	result := httpValidator(request)
	assert.Equal(t, "The testFile field file extension csv is invalid", result.Get("testFile"))
}

func TestRequestValidatorWrongEnvVars(t *testing.T) {
	requestFiles := map[string]string{
		"testFile": "testdata/valid/loadtest.jmx",
		"testData": "testdata/valid/loadtest.jmx",
		"envVars":  "testdata/valid/loadtest.jmx",
	}

	distributedPods := "2"

	request, err := buildMocFormReq(requestFiles, distributedPods, "JMeter")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	result := httpValidator(request)
	assert.Equal(t, "The envVars field file extension jmx is invalid", result.Get("envVars"))
	assert.Equal(t, "The testData field file extension jmx is invalid", result.Get("testData"))
}

func buildMocFormReq(requestFiles map[string]string, distributedPods, ltType string) (*http.Request, error) {
	request, err := createRequestBody(requestFiles, distributedPods, ltType)
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
