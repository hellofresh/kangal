package locust

import (
	"errors"
	"time"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	// ErrRequireMinOneDistributedPod spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

//BuildLoadTestSpec validates input and returns valid LoadTestSpec
func BuildLoadTestSpec(
	config Config,
	overwrite bool,
	distributedPods int32,
	tags loadTestV1.LoadTestTags,
	testFileStr, envVarsStr, targetURL string,
	duration time.Duration,
) (loadTestV1.LoadTestSpec, error) {
	lt := loadTestV1.LoadTestSpec{}
	if distributedPods <= int32(0) {
		return lt, ErrRequireMinOneDistributedPod
	}
	if testFileStr == "" {
		return lt, ErrRequireTestFile
	}

	imageName := defaultImage
	imageTag := defaultImageTag

	// this is to ensure backward compatibility
	if config.Image != "" && config.ImageName == "" {
		config.ImageName = config.Image
	}

	// Use environment variable config if available
	if config.ImageName != "" {
		imageName = config.ImageName
	}
	if config.ImageTag != "" {
		imageTag = config.ImageTag
	}

	return loadTestV1.NewSpec(
		loadTestV1.LoadTestTypeLocust,
		overwrite,
		distributedPods,
		tags,
		testFileStr,
		"",
		envVarsStr,
		loadTestV1.ImageDetails{Image: imageName, Tag: imageTag},
		loadTestV1.ImageDetails{Image: imageName, Tag: imageTag},
		targetURL,
		duration,
	), nil
}
