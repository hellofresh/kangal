package jmeter

import (
	"errors"
	"time"

	"github.com/docker/distribution/reference"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	// ErrRequireMinOneDistributedPod JMeter spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

//BuildLoadTestSpec validates input and returns valid LoadTestSpec for JMeter backend provider
func BuildLoadTestSpec(config Config, overwrite bool, distributedPods int32, tags loadTestV1.LoadTestTags, testFileStr, testDataStr, envVarsStr string, masterImageRef, workerImageRef reference.NamedTagged) (loadTestV1.LoadTestSpec, error) {
	lt := loadTestV1.LoadTestSpec{}
	// JMeter backend provider needs full spec: from number of distributed pods to envVars
	if distributedPods <= int32(0) {
		return lt, ErrRequireMinOneDistributedPod
	}
	if testFileStr == "" {
		return lt, ErrRequireTestFile
	}

	masterImageName := defaultMasterImageName
	masterImageTag := defaultMasterImageTag
	workerImageName := defaultWorkerImageName
	workerImageTag := defaultWorkerImageTag

	// Use environment variable config if available
	if config.MasterImageName != "" {
		masterImageName = config.MasterImageName
	}
	if config.MasterImageTag != "" {
		masterImageTag = config.MasterImageTag
	}
	if config.WorkerImageName != "" {
		workerImageName = config.WorkerImageName
	}
	if config.WorkerImageTag != "" {
		workerImageTag = config.WorkerImageTag
	}

	// Use loadtest data received from proxy if available
	if masterImageRef != nil {
		masterImageName = masterImageRef.Name()
		masterImageTag = masterImageRef.Tag()
	}
	if workerImageRef != nil {
		workerImageName = workerImageRef.Name()
		workerImageTag = workerImageRef.Tag()
	}

	return loadTestV1.NewSpec(loadTestV1.LoadTestTypeJMeter, overwrite, distributedPods, tags, testFileStr, testDataStr, envVarsStr, loadTestV1.ImageDetails{Image: masterImageName, Tag: masterImageTag}, loadTestV1.ImageDetails{Image: workerImageName, Tag: workerImageTag}, "", time.Duration(0)), nil
}
