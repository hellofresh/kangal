package locust

import (
	"testing"

	batchV1 "k8s.io/api/batch/v1"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetLoadTestStatusFromJobs(t *testing.T) {
	var scenarios = []struct {
		MasterJob *batchV1.Job
		WorkerJob *batchV1.Job
		Expected  loadtestV1.LoadTestPhase
	}{
		{ // Starting
			MasterJob: &batchV1.Job{},
			WorkerJob: &batchV1.Job{},
			Expected:  loadtestV1.LoadTestStarting,
		},
		{ // One master, two workers, all running
			MasterJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Active: int32(1),
				},
			},
			WorkerJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Active: int32(2),
				},
			},
			Expected: loadtestV1.LoadTestRunning,
		},
		{ // One worker failed
			MasterJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Active: int32(1),
				},
			},
			WorkerJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Active: int32(1),
					Failed: int32(1),
				},
			},
			Expected: loadtestV1.LoadTestErrored,
		},
		{ // Master failed, workers running
			MasterJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Failed: int32(1),
				},
			},
			WorkerJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Active: int32(2),
				},
			},
			Expected: loadtestV1.LoadTestErrored,
		},
		{ // Workers finished, master running
			MasterJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Active: int32(1),
				},
			},
			WorkerJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Succeeded: int32(2),
				},
			},
			Expected: loadtestV1.LoadTestRunning,
		},
		{ // Master finished, workers running, unexpected scenario
			MasterJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Succeeded: int32(1),
				},
			},
			WorkerJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Active: int32(2),
				},
			},
			Expected: loadtestV1.LoadTestRunning,
		},
		{ // Both succeeded
			MasterJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Succeeded: int32(1),
				},
			},
			WorkerJob: &batchV1.Job{
				Status: batchV1.JobStatus{
					Succeeded: int32(2),
				},
			},
			Expected: loadtestV1.LoadTestFinished,
		},
	}

	for _, scenario := range scenarios {
		actual := getLoadTestStatusFromJobs(scenario.MasterJob, scenario.WorkerJob)
		assert.Equal(t, scenario.Expected, actual)
	}
}
