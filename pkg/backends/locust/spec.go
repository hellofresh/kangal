package locust

import (
	"errors"
	"time"

	"github.com/docker/distribution/reference"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	// ErrRequireMinOneDistributedPod spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

//BuildLoadTestSpec validates input and returns valid LoadTestSpec
func BuildLoadTestSpec(overwrite bool, distributedPods int32, tags loadTestV1.LoadTestTags, testFileStr, envVarsStr, targetURL string, duration time.Duration, masterImageRef reference.NamedTagged) (loadTestV1.LoadTestSpec, error) {
	lt := loadTestV1.LoadTestSpec{}
	if distributedPods <= int32(0) {
		return lt, ErrRequireMinOneDistributedPod
	}
	if testFileStr == "" {
		return lt, ErrRequireTestFile
	}
	// Use defaults if unspecified
	masterImage := loadTestV1.ImageDetails{Image: defaultImage, Tag: defaultImageTag}
	if masterImageRef != nil {
		masterImage = loadTestV1.ImageDetails{
			Image: masterImageRef.Name(),
			Tag:   masterImageRef.Tag(),
		}
	}
	return loadTestV1.NewSpec(loadTestV1.LoadTestTypeLocust, overwrite, distributedPods, tags, testFileStr, "", envVarsStr, masterImage, loadTestV1.ImageDetails{}, targetURL, duration), nil
}
