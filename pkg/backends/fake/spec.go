package fake

import (
	"time"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

//BuildLoadTestSpec returns LoadTestSpec for Fake backend provider
func BuildLoadTestSpec(tags loadTestV1.LoadTestTags, overwrite bool) (loadTestV1.LoadTestSpec, error) {
	// in general Fake backend provider doesn't need any fields except overwrite flag
	return loadTestV1.NewSpec(loadTestV1.LoadTestTypeFake, overwrite, 1, tags, "", "", "", loadTestV1.ImageDetails{Image: sleepImage, Tag: imageTag}, loadTestV1.ImageDetails{}, "", time.Duration(0)), nil
}
