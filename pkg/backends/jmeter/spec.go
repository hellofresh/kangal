package jmeter

import (
	"errors"
	"time"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	// ErrRequireMinOneDistributedPod JMeter spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

//BuildLoadTestSpec validates input and returns valid LoadTestSpec for JMeter backend provider
func BuildLoadTestSpec(overwrite bool, distributedPods int32, testFileStr, testDataStr, envVarsStr string) (loadTestV1.LoadTestSpec, error) {
	lt := loadTestV1.LoadTestSpec{}
	// JMeter backend provider needs full spec: from number of distributed pods to envVars
	if distributedPods <= int32(0) {
		return lt, ErrRequireMinOneDistributedPod
	}
	if testFileStr == "" {
		return lt, ErrRequireTestFile
	}
	return loadTestV1.NewSpec(loadTestV1.LoadTestTypeJMeter, overwrite, distributedPods, testFileStr, testDataStr, envVarsStr, loadTestV1.ImageDetails{}, loadTestV1.ImageDetails{}, "", time.Duration(0)), nil
}
