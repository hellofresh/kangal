package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/thedevsaddam/govalidator"
	"go.uber.org/zap"

	"github.com/hellofresh/kangal/pkg/backends"
	"github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	backendType     = "type"
	overwrite       = "overwrite"
	distributedPods = "distributedPods"
	tags            = "tags"
	testFile        = "testFile"
	testData        = "testData"
	envVars         = "envVars"
	targetURL       = "targetURL"
	duration        = "duration"
	loadTestID      = "id"
	masterImage     = "masterImage"
	workerImage     = "workerImage"
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
		"targetURL":       []string{"http"},
		"duration":        []string{"duration"},
	}

	opts := govalidator.Options{
		Request:         r,     // request object
		Rules:           rules, // rules map
		RequiredDefault: false, // all the field to be pass the rules,
	}

	v := govalidator.New(opts)
	return v.Validate()
}

func fromHTTPRequestToListOptions(r *http.Request) (*kubernetes.ListOptions, error) {
	opt := kubernetes.ListOptions{}
	params := r.URL.Query()

	// Build tags filter.
	if tagsString := params.Get("tags"); tagsString != "" {
		tags, err := apisLoadTestV1.LoadTestTagsFromString(tagsString)
		if err != nil {
			return nil, err
		}

		opt.Tags = tags
	}

	// Build limit filter.
	if limitVal := params.Get("limit"); limitVal != "" {
		limit, err := strconv.ParseInt(limitVal, 10, 64)
		if err != nil {
			return nil, err
		}

		opt.Limit = limit
	}

	// Build continue.
	opt.Continue = params.Get("continue")

	return &opt, nil
}

// fromHTTPRequestToLoadTestSpec creates a load test spec from HTTP request
func fromHTTPRequestToLoadTestSpec(r *http.Request, cfg backends.Config, logger *zap.Logger) (apisLoadTestV1.LoadTestSpec, error) {
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

	tagList, err := getTags(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", "tags"), zap.String("tags", tags), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %q from request: %w", tags, err)
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

	turl, err := getTargetURL(r)
	if err != nil {
		logger.Debug("Bad value", zap.String("field", targetURL), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %q from request: %w", targetURL, err)
	}

	dur, err := getDuration(r)
	if err != nil {
		logger.Debug("Bad value", zap.String("field", duration), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %q from request: %w", duration, err)
	}

	masterImageRef, err := getMasterImageRef(r)
	if err != nil {
		logger.Debug("Bad value", zap.String("field", masterImage), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %q from request: %w", masterImage, err)
	}

	workerImageRef, err := getWorkerImageRef(r)
	if err != nil {
		logger.Debug("Bad value", zap.String("field", workerImage), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %q from request: %w", workerImage, err)
	}

	return backends.BuildLoadTestSpecByBackend(ltType, o, dp, tagList, tf, td, ev, turl, dur, masterImageRef, workerImageRef)
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

func getTargetURL(r *http.Request) (string, error) {
	return r.FormValue(targetURL), nil
}

func getDuration(r *http.Request) (time.Duration, error) {
	val := r.FormValue(duration)

	if "" == val {
		return time.Duration(0), nil
	}

	return time.ParseDuration(val)
}

func getMasterImageRef(r *http.Request) (reference.NamedTagged, error) {
	val := r.FormValue(masterImage)
	return parseImageRef(val)
}

func getWorkerImageRef(r *http.Request) (reference.NamedTagged, error) {
	val := r.FormValue(workerImage)
	return parseImageRef(val)
}

func parseImageRef(val string) (reference.NamedTagged, error) {
	if val == "" {
		return nil, nil
	}

	image, err := reference.ParseNormalizedNamed(val)
	if err != nil {
		return nil, err
	}

	ref := reference.TagNameOnly(image).(reference.NamedTagged)
	return ref, nil
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

func getTags(r *http.Request) (apisLoadTestV1.LoadTestTags, error) {
	return apisLoadTestV1.LoadTestTagsFromString(r.FormValue(tags))
}
