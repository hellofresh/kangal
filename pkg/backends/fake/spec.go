package fake

import loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"

//BuildLoadTestSpec returns LoadTestSpec for Fake backend provider
func BuildLoadTestSpec(overwrite bool) (loadTestV1.LoadTestSpec, error) {
	// in general Fake backend provider doesn't need any fields except overwrite flag
	return loadTestV1.NewSpec(loadTestV1.LoadTestTypeFake, overwrite, 1, "", "", "", loadTestV1.ImageDetails{Image: sleepImage, Tag: imageTag}, loadTestV1.ImageDetails{}), nil
}
