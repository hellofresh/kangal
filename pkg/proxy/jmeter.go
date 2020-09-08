package proxy

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/technosophos/moniker"
	"github.com/thedevsaddam/govalidator"
	"go.uber.org/zap"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	backendType     = "type"
	distributedPods = "distributedPods"
	testFile        = "testFile"
	testData        = "testData"
	envVars         = "envVars"
	loadTestID      = "id"
)

// JMeter is a type of Loadtest
type JMeter struct {
	Spec   *apisLoadTestV1.LoadTestSpec
	Logger *zap.Logger
}

var (
	// ErrRequiredJMeterType error on LoadTest type if not of type JMeter
	ErrRequiredJMeterType = errors.New("LoadTest of type JMeter is required")
	// ErrRequireMinOneDistributedPod JMeter spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
	// ErrEmptySpec JMeter struct must have a non nil Spec
	ErrEmptySpec = errors.New("JMeter requires a Spec field to validate")
)

// NewJMeterLoadTest returns a loadtest with the correct under lying loadtest type
func NewJMeterLoadTest(spec *apisLoadTestV1.LoadTestSpec, logger *zap.Logger) (*JMeter, error) {
	jm := &JMeter{
		Logger: logger,
		Spec:   spec,
	}

	return jm, jm.validate()
}

//RequestValidator validates request body
func httpValidator(r *http.Request) url.Values {
	rules := govalidator.MapData{
		"type":            []string{"required", "in:JMeter,Fake"},
		"distributedPods": []string{"required", "numeric_between:1,"},
		"file:testFile":   []string{"required", "ext:jmx"},
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

// FromHTTPRequestToJMeter creates a JMeter loadtest struct
func FromHTTPRequestToJMeter(r *http.Request, ltType apisLoadTestV1.LoadTestType, logger *zap.Logger) (*apisLoadTestV1.LoadTestSpec, error) {
	if e := httpValidator(r); len(e) > 0 {
		logger.Debug("User request validation failed", zap.Any("errors", e))
		return nil, fmt.Errorf(e.Encode())
	}

	spec := &apisLoadTestV1.LoadTestSpec{
		Type: ltType,
	}

	n, err := getDistributedPods(r)
	if err != nil {
		logger.Debug("Bad value: ", zap.String("field", distributedPods), zap.Int32("value", n), zap.Error(err))
		return nil, fmt.Errorf("bad %q value: should be integer", distributedPods)
	}
	spec.DistributedPods = &n

	tf, err := getTestFile(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", testFile), zap.Error(err))
		return nil, fmt.Errorf("error getting %q from request: %w", testFile, err)
	}
	spec.TestFile = tf

	td, err := getTestData(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", testData), zap.Error(err))
		return nil, fmt.Errorf("error getting %q from request: %w", testData, err)
	}
	spec.TestData = td

	ev, err := getEnvVars(r)
	if err != nil {
		logger.Debug("Could not get file from request", zap.String("file", envVars), zap.Error(err))
		return nil, fmt.Errorf("error getting %q from request: %w", envVars, err)
	}
	spec.EnvVars = ev

	return spec, nil
}

//ValidateRequest validates request body
func (jm *JMeter) validate() error {
	if jm.Spec == nil {
		return ErrEmptySpec
	}

	// we use Fake loadtest type to simulate JMeter loadtest creation
	if jm.Spec.Type != apisLoadTestV1.LoadTestTypeJMeter && jm.Spec.Type != apisLoadTestV1.LoadTestTypeFake {
		return ErrRequiredJMeterType
	}

	if jm.Spec.DistributedPods == nil || *jm.Spec.DistributedPods <= int32(0) {
		return ErrRequireMinOneDistributedPod
	}

	if jm.Spec.TestFile == "" {
		return ErrRequireTestFile
	}

	return nil
}

// Hash returns the hash of a JMeter loadtest
func (jm *JMeter) Hash() string {
	hasher := sha1.New()
	hasher.Write([]byte(jm.Spec.TestFile))
	return hex.EncodeToString(hasher.Sum(nil))
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

	stringTestData, err := FileToString(td)
	if err != nil {
		return "", err
	}

	return stringTestData, nil
}

func getDistributedPods(r *http.Request) (int32, error) {
	nn := r.FormValue(distributedPods)
	dn, err := strconv.Atoi(nn)
	if err != nil {
		return 0, err
	}

	return int32(dn), nil
}

// ToLoadTest converts cr structure to LoadTest
func (jm *JMeter) ToLoadTest() *apisLoadTestV1.LoadTest {
	loadTest := &apisLoadTestV1.LoadTest{
		Spec: *jm.Spec,
	}

	generatedName := moniker.New().NameSep("-")

	loadTest.Name = "loadtest-" + generatedName

	loadTest.Labels = map[string]string{
		"test-file-hash": jm.Hash(),
	}

	return loadTest
}
