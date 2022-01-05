package proxy

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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

var (
	// ErrFileToStringEmpty is the error returned when the defined users file is empty
	ErrFileToStringEmpty = errors.New("file is empty")
	// ErrWrongFileFormat is the error returned when the defined users file is empty
	ErrWrongFileFormat = errors.New("file format is not supported")
	// ErrWrongURLFormat is the error returned when the targetURL is not containing scheme
	ErrWrongURLFormat = errors.New("invalid URL format")
	// ErrWrongImageFormat is the error returned when the docker image is in wrong format
	ErrWrongImageFormat = errors.New("invalid image format")
	// ErrEmptyType is the error returned when there's no loadtest type provided
	ErrEmptyType = errors.New("loadtest type is empty")

	testFileFormats     = []string{"jmx", "py", "json", "toml"}
	testDataFileFormats = []string{"csv", "protoset"}
)

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
	o, err := getOverwrite(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", overwrite), zap.Bool("value", o), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("bad %s value: should be bool", overwrite)
	}

	lt, err := getLoadTestType(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", backendType), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", backendType, err)
	}

	dp, err := getDistributedPods(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", distributedPods), zap.Int32("value", dp), zap.Error(err))
		return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("bad %s value: should be integer", distributedPods)
	}

	tagList, err := getTags(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", tags), zap.String("tags", tags), zap.Error(err))
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
		mi, err = getImage(r, masterImage)
		if err != nil {
			logger.Debug("Bad value", zap.String("field", masterImage), zap.Error(err))
			return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", masterImage, err)
		}
		wi, err = getImage(r, workerImage)
		if err != nil {
			logger.Debug("Bad value", zap.String("field", workerImage), zap.Error(err))
			return apisLoadTestV1.LoadTestSpec{}, fmt.Errorf("error getting %s from request: %w", workerImage, err)
		}
	}

	return apisLoadTestV1.LoadTestSpec{
		Type:            lt,
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
	stringEnv, fileType, err := getFileFromHTTP(r, envVars)
	if err != nil {
		//this means there was no envVars file specified and we can ignore this error because envVars is optional
		if err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, err
	}
	if fileType != "csv" {
		return nil, ErrWrongFileFormat
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
		//this means there was no testData file specified and we can ignore this error because testData is optional
		if err == http.ErrMissingFile {
			return "", nil
		}
		return "", err
	}

	for _, f := range testDataFileFormats {
		if fileType == f {
			if fileType == "csv" {
				err = checkCsvFile(stringTestData)
				if err != nil {
					return "", err
				}
			}
			return stringTestData, nil
		}
	}

	return "", ErrWrongFileFormat
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
	if err != nil {
		return "", err
	}

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

func getLoadTestType(r *http.Request) (apisLoadTestV1.LoadTestType, error) {
	ltType := r.FormValue(backendType)
	if ltType == "" {
		return "", ErrEmptyType
	}

	return apisLoadTestV1.LoadTestType(ltType), nil
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
	targetURL := r.FormValue(targetURL)

	if targetURL == "" {
		return "", nil
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}

	if u.Scheme == "" || u.Host == "" {
		return "", ErrWrongURLFormat
	}
	return targetURL, nil
}

func getDuration(r *http.Request) (time.Duration, error) {
	val := r.FormValue(duration)

	if "" == val {
		return time.Duration(0), nil
	}

	return time.ParseDuration(val)
}

func getImage(r *http.Request, role string) (apisLoadTestV1.ImageDetails, error) {
	imageStr := r.FormValue(role)

	imageRegex := regexp.MustCompile("^.*:.*$|^$")
	match := imageRegex.Match([]byte(imageStr))
	if !match {
		return apisLoadTestV1.ImageDetails{}, ErrWrongImageFormat
	}

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
	}, nil
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
