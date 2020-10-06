package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thedevsaddam/govalidator"
	"go.uber.org/zap"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	backendType     = "type"
	overwrite       = "overwrite"
	distributedPods = "distributedPods"
	testFile        = "testFile"
	testData        = "testData"
	envVars         = "envVars"
	loadTestID      = "id"
)

//httpValidator validates request body
func httpValidator(r *http.Request) url.Values {
	rules := govalidator.MapData{
		"type":            []string{"required"},
		"overwrite":       []string{"in:1,True,true,t,T,TRUE,0,False,false,f,F,FALSE"},
		"distributedPods": []string{"numeric_between:1,"},
		"file:testFile":   []string{"ext:jmx,py"},
		"file:envVars":    []string{"ext:csv"},
		"file:testData":   []string{"ext:csv"},
	}

	opts := govalidator.Options{
		Request:         r,     // request object
		Rules:           rules, // rules map
		RequiredDefault: false, // all the field to be pass the rules,
	}

	v := govalidator.New(opts)
	return v.Validate()
}

// fromHTTPRequestToLoadTestSpec creates a load test spec from HTTP request
func fromHTTPRequestToLoadTestSpec(r *http.Request, logger *zap.Logger) (apisLoadTestV1.LoadTestSpec, error) {
	ltType := getLoadTestType(r)

	if e := httpValidator(r); len(e) > 0 {
		logger.Debug("User request validation failed", zap.Any("errors", e))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf(e.Encode())
	}

	o, err := getOverwrite(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", overwrite), zap.Bool("value", o), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("bad %q value: should be bool", overwrite)
	}

	dp, err := getDistributedPods(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", "distributedPods"), zap.Int32("value", dp), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("bad %q value: should be integer", distributedPods)
	}

	tf, err := getTestFile(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", testFile), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %q from request: %w", testFile, err)
	}

	td, err := getTestData(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", testData), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %q from request: %w", testData, err)
	}

	ev, err := getEnvVars(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", envVars), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %q from request: %w", envVars, err)
	}

	return apisLoadTestV1.BuildLoadTestSpec(ltType, o, dp, tf, td, ev)
}

func getEnvVars(r *http.Request) (string, error) {
	return getFileFromHTTP(r, envVars)
}

func getTestData(r *http.Request) (string, error) {
	return getFileFromHTTP(r, testData)
}

func getTestFile(r *http.Request) (string, error) {
	return getFileFromHTTP(r, testFile)
}

func getFileFromHTTP(r *http.Request, file string) (string, error) {
	td, _, err := r.FormFile(file)
	if err != nil {
		// this means there was no file specified and we should ignore the error
		if err == http.ErrMissingFile {
			return "", nil
		}

		return "", err
	}

	stringTestData, err := fileToString(td)
	if err != nil {
		return "", err
	}

	return stringTestData, nil
}

func getOverwrite(r *http.Request) (bool, error) {
	o := r.FormValue(overwrite)

	if o == "" {
		return false, nil
	}

	overwrite, err := strconv.ParseBool(o)
	if err != nil {
		return false, err
	}

	return overwrite, nil
}

func getDistributedPods(r *http.Request) (int32, error) {
	nn := r.FormValue(distributedPods)
	dn, err := strconv.Atoi(nn)
	if err != nil {
		return 0, err
	}

	return int32(dn), nil
}

//fileToString converts file to string
func fileToString(f io.ReadCloser) (string, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(f); err != nil {
		return "", err
	}

	defer f.Close()
	s := buf.String()

	if len(s) == 0 {
		return "", ErrFileToStringEmpty
	}

	return s, nil
}
