package v1

import "errors"

var (
	// ErrInvalidLoadTestType error on LoadTest type if not of allowed types
	ErrInvalidLoadTestType = errors.New("invalid LoadTest type")
	// ErrRequireMinOneDistributedPod JMeter spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

//BuildLoadTestSpec initialize spec for LoadTest custom resource
func BuildLoadTestSpec(loadTestType LoadTestType, distributedPods int32, testFileStr, testDataStr, envVarsStr string) (LoadTestSpec, error) {
	lt := LoadTestSpec{}

	if loadTestType != LoadTestTypeJMeter && loadTestType != LoadTestTypeFake {
		return lt, ErrInvalidLoadTestType
	}

	if distributedPods <= int32(0) {
		return lt, ErrRequireMinOneDistributedPod
	}

	if testFileStr == "" {
		return lt, ErrRequireTestFile
	}

	lt.Type = loadTestType
	lt.DistributedPods = &distributedPods
	lt.TestFile = testFileStr
	lt.TestData = testDataStr
	lt.EnvVars = envVarsStr

	return lt, nil
}
