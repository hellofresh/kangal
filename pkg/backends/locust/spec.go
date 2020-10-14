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
func BuildLoadTestSpec(overwrite bool, distributedPods int32, testFileStr, envVarsStr, targetURL string, duration time.Duration) (loadTestV1.LoadTestSpec, error) {
	lt := loadTestV1.LoadTestSpec{}
	if distributedPods <= int32(0) {
		return lt, ErrRequireMinOneDistributedPod
	}
	if testFileStr == "" {
		return lt, ErrRequireTestFile
	}
	return loadTestV1.NewSpec(loadTestV1.LoadTestTypeLocust, overwrite, distributedPods, testFileStr, "", envVarsStr, loadTestV1.ImageDetails{}, loadTestV1.ImageDetails{}, targetURL, duration), nil
}
