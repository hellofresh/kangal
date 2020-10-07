package v1

import (
	"errors"
	"time"
)

var (
	// ErrInvalidLoadTestType error on LoadTest type if not of allowed types
	ErrInvalidLoadTestType = errors.New("invalid LoadTest type")
	// ErrRequireMinOneDistributedPod JMeter spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

//BuildLoadTestSpec initialize spec for LoadTest custom resource
func BuildLoadTestSpec(loadTestType LoadTestType, overwrite bool, distributedPods int32, testFileStr, testDataStr, envVarsStr string, targetURL string, duration time.Duration) (LoadTestSpec, error) {
	lt := LoadTestSpec{}

	if false == HasLoadTestType(loadTestType) {
		return lt, ErrInvalidLoadTestType
	}

	if distributedPods <= int32(0) {
		return lt, ErrRequireMinOneDistributedPod
	}

	if testFileStr == "" {
		return lt, ErrRequireTestFile
	}

	lt.Type = loadTestType
	lt.Overwrite = overwrite
	lt.DistributedPods = &distributedPods
	lt.TestFile = testFileStr
	lt.TestData = testDataStr
	lt.EnvVars = envVarsStr
	lt.TargetURL = targetURL
	lt.Duration = duration

	return lt, nil
}
