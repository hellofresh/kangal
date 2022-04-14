package v1

import (
	"errors"
	"time"
)

var (
	// ErrTagMissingLabel indicates that tag label is missing.
	ErrTagMissingLabel = errors.New("missing tag label")
	// ErrTagMissingValue indicates that tag value is missing.
	ErrTagMissingValue = errors.New("missing tag value")
	// ErrTagValueMaxLengthExceeded indicates that tag value is too long.
	ErrTagValueMaxLengthExceeded = errors.New("tag value is too long")
)

//NewSpec initialize spec for LoadTest custom resource
func NewSpec(loadTestType LoadTestType, overwrite bool, distributedPods int32, tags LoadTestTags, testFile, testData []byte, envVars map[string]string, masterConfig, workerConfig ImageDetails, targetURL string, duration time.Duration) LoadTestSpec {
	return LoadTestSpec{
		Type:            loadTestType,
		Overwrite:       overwrite,
		MasterConfig:    masterConfig,
		WorkerConfig:    workerConfig,
		DistributedPods: &distributedPods,
		Tags:            tags,
		TestFile:        testFile,
		TestData:        testData,
		EnvVars:         envVars,
		TargetURL:       targetURL,
		Duration:        duration,
	}
}
