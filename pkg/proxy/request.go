package proxy

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thedevsaddam/govalidator"
	"go.uber.org/zap"

	"github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	backendType     = "type"
	overwrite       = "overwrite"
	masterImage     = "masterImage"
	workerImage     = "workerImage"
	distributedPods = "distributedPods"
	tags            = "tags"
	testFile        = "testFile"
	testData        = "testData"
	envVars         = "envVars"
	targetURL       = "targetURL"
	duration        = "duration"
	loadTestID      = "id"
	workerPodID     = "worker"
)

var testFileFormats = []string{"jmx", "py", "json", "toml"}

//httpValidator validates request body
func httpValidator(r *http.Request) url.Values {
	rules := govalidator.MapData{
		"type":          []string{"required"},
		"masterImage":   []string{"regex:^.*:.*$|^$"},
		"workerImage":   []string{"regex:^.*:.*$|^$"},
		"file:testData": []string{"ext:csv,protoset"},
		"targetURL":     []string{"http"},
	}

	opts := govalidator.Options{
		Request:         r,     // request object
		Rules:           rules, // rules map
		RequiredDefault: false, // all the field to be pass the rules,
	}

	v := govalidator.New(opts)
	return v.Validate()
}

func fromHTTPRequestToListOptions(r *http.Request, maxListLimit int64) (*kubernetes.ListOptions, error) {
	opt := kubernetes.ListOptions{
		Limit: maxListLimit,
	}
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

		if limit > maxListLimit {
			return nil, fmt.Errorf("limit value is too big, max possible value is %d", maxListLimit)
		}

		opt.Limit = limit
	}

	// Build phase filter.
	phase, err := apisLoadTestV1.LoadTestPhaseFromString(params.Get("phase"))
	if err != nil {
		return nil, err
	}
	opt.Phase = phase

	// Build continue.
	opt.Continue = params.Get("continue")

	return &opt, nil
}

// fromHTTPRequestToLoadTestSpec creates a load test spec from HTTP request
func fromHTTPRequestToLoadTestSpec(r *http.Request, logger *zap.Logger, allowedCustomImages bool) (apisLoadTestV1.LoadTestSpec, error) {
	if e := httpValidator(r); len(e) > 0 {
		logger.Debug("User request validation failed", zap.Any("errors", e))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf(e.Encode())
	}

	o, err := getOverwrite(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", overwrite), zap.Bool("value", o), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("bad %s value: should be bool", overwrite)
	}

	dp, err := getDistributedPods(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", "distributedPods"), zap.Int32("value", dp), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("bad %s value: should be integer", distributedPods)
	}

	tagList, err := getTags(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", "tags"), zap.String("tags", tags), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", tags, err)
	}

	tf, err := getTestFile(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", testFile), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", testFile, err)
	}

	td, err := getTestData(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", testData), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", testData, err)
	}

	ev, err := getEnvVars(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", envVars), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", envVars, err)
	}

	turl, err := getTargetURL(r)
	if err != nil {
		logger.Debug("Bad value", zap.String("field", targetURL), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", targetURL, err)
	}

	dur, err := getDuration(r)
	if err != nil {
		logger.Debug("Bad value", zap.String("field", duration), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", duration, err)
	}

	mi := apisLoadTestV1.ImageDetails{
		Image: "",
		Tag:   "",
	}
	wi := apisLoadTestV1.ImageDetails{
		Image: "",
		Tag:   "",
	}
	if allowedCustomImages {
		mi = getImage(r, masterImage)
		wi = getImage(r, workerImage)
	}

	return apisLoadTestV1.LoadTestSpec{
		Type:            getLoadTestType(r),
		Overwrite:       o,
		MasterConfig:    mi,
		WorkerConfig:    wi,
		DistributedPods: &dp,
		Tags:            tagList,
		TestFile:        tf,
		TestData:        td,
		EnvVars:         ev,
		TargetURL:       turl,
		Duration:        dur,
	}, nil
}

func getEnvVars(r *http.Request) (map[string]string, error) {
	stringEnv, _, err := getFileFromHTTP(r, envVars)
	if err != nil {
		return nil, err
	}
	s, err := ReadEnvs(stringEnv)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func getTestData(r *http.Request) (string, error) {
	stringTestData, fileType, err := getFileFromHTTP(r, testData)
	if err != nil {
		return "", err
	}

	if fileType == "csv" {
		err = checkCsvFile(stringTestData)
		if err != nil {
			return "", err
		}
	}

	return stringTestData, nil
}

func checkCsvFile(s string) error {
	reader := csv.NewReader(strings.NewReader(s))
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func getTestFile(r *http.Request) (string, error) {
	content, fileType, err := getFileFromHTTP(r, testFile)

	for _, f := range testFileFormats {
		if fileType == f {
			return content, err
		}
	}
	return "", ErrWrongFileFormat
}

func getFileFromHTTP(r *http.Request, file string) (string, string, error) {
	td, meta, err := r.FormFile(file)
	if err != nil {
		// this means there was no file specified and we should ignore the error
		if err == http.ErrMissingFile {
			return "", "", nil
		}

		return "", "", err
	}

	stringTestData, err := fileToString(td)
	if err != nil {
		return "", "", err
	}

	return stringTestData, getTypeFromName(meta.Filename), nil
}

func getTypeFromName(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
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
	dn, err := strconv.ParseInt(nn, 10, 32)
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

func getImage(r *http.Request, role string) apisLoadTestV1.ImageDetails {

	imageStr := r.FormValue(role)

	imgName := ""
	imgTag := ""

	// Gen image url colon and slash struct
	structImgChars := ":/"
	structImgURI := ""
	for _, c := range imageStr {

		if strings.Contains(structImgChars, string(c)) {
			structImgURI += string(c)
		}
	}

	if structImgURI == "" {
		// Format: image
		imgName = imageStr
		imgTag = ""
	}

	if structImgURI == ":" {
		// Format: image:tag
		imgName = strings.Split(imageStr, ":")[0]
		imgTag = strings.Split(imageStr, ":")[1]
	}

	if structImgURI == "/" {
		// Format: registry/image
		imgName = imageStr
		imgTag = ""
	}

	if structImgURI == "/:" {
		// Format: registry/image:tag
		imgName = strings.Split(imageStr, ":")[0]
		imgTag = strings.Split(imageStr, ":")[1]
	}

	if structImgURI == "://" {
		// Format: host:port/registry/image
		imgName = imageStr
		imgTag = ""
	}

	if structImgURI == "://:" {
		// Format: host:port/registry/image:tag
		imgName = strings.Split(imageStr, ":")[0] + ":" + strings.Split(imageStr, ":")[1]
		imgTag = strings.Split(imageStr, ":")[2]
	}

	if structImgURI == "//:" {
		// Format: host/registry/image:tag
		imgName = strings.Split(imageStr, ":")[0]
		imgTag = strings.Split(imageStr, ":")[1]
	}

	return apisLoadTestV1.ImageDetails{
		Image: imgName,
		Tag:   imgTag,
	}
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
