package proxy

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
			"Invalid type",
			"1",
			"type",
			"IncorrectType",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			"The type field must be one of JMeter, Fake",
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

func TestHTTPValidatorWithTimeout(t *testing.T) {
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
			"Invalid type",
			"1",
			"type",
			"IncorrectType",
			map[string]string{
				"testFile": "testdata/valid/loadtest.jmx",
			},
			"The type field must be one of JMeter, Fake",
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
				assert.Equal(t, ctx.Err().Error(), "context deadline exceeded")
			}

		})
	}
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
