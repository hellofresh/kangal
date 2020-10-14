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
	// ErrTagMissingLabel indicates that tag label is missing.
	ErrTagMissingLabel = errors.New("missing tag label")
	// ErrTagMissingValue indicates that tag value is missing.
	ErrTagMissingValue = errors.New("missing tag value")
	// ErrTagValueMaxLengthExceeded indicates that tag value is too long.
	ErrTagValueMaxLengthExceeded = errors.New("tag value is too long")
)

//NewSpec initialize spec for LoadTest custom resource
func NewSpec(loadTestType LoadTestType, overwrite bool, distributedPods int32, tags LoadTestTags, testFileStr, testDataStr, envVarsStr string, masterConfig, workerConfig ImageDetails, targetURL string, duration time.Duration) LoadTestSpec {
	return LoadTestSpec{
		Type:            loadTestType,
		Overwrite:       overwrite,
		MasterConfig:    masterConfig,
		WorkerConfig:    workerConfig,
		DistributedPods: &distributedPods,
		Tags:            tags,
		TestFile:        testFileStr,
		TestData:        testDataStr,
		EnvVars:         envVarsStr,
		TargetURL:       targetURL,
		Duration:        duration,
	}
}
