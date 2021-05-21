package ghz

import (
	"testing"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
	batchV1 "k8s.io/api/batch/v1"
)

func TestGetLoadTestStatusFromJobs(t *testing.T) {
	var scenarios = []struct {
		NumberActive    int32
		NumberFailed    int32
		NumberSucceeded int32
		ExpectedPhase   loadTestV1.LoadTestPhase
	}{
		{0, 0, 0, loadTestV1.LoadTestStarting},
		{1, 0, 0, loadTestV1.LoadTestRunning},
		{2, 0, 0, loadTestV1.LoadTestRunning},
		{0, 1, 0, loadTestV1.LoadTestErrored},
		{1, 1, 0, loadTestV1.LoadTestErrored},
		{2, 1, 0, loadTestV1.LoadTestErrored},
		{2, 0, 1, loadTestV1.LoadTestRunning},
		{1, 0, 1, loadTestV1.LoadTestRunning},
		{0, 0, 1, loadTestV1.LoadTestFinished},
		{0, 1, 1, loadTestV1.LoadTestErrored},
		{1, 1, 1, loadTestV1.LoadTestErrored},
	}

	for _, scenario := range scenarios {
		job := &batchV1.Job{
			Status: batchV1.JobStatus{
				Active:    scenario.NumberActive,
				Failed:    scenario.NumberFailed,
				Succeeded: scenario.NumberSucceeded,
			},
		}
		actual := determineLoadTestStatusFromJobs(job)
		assert.Equal(t, scenario.ExpectedPhase, actual)
	}
}
