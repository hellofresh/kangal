package jmeter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestCheckForTimeout(t *testing.T) {
	// subtract 10 minutes from the current time
	now := metaV1.Now()
	maxWaitTime := metaV1.Now().Add(MaxWaitTimeForPods * -1)
	past := metaV1.Time{Time: maxWaitTime}

	var tests = []struct {
		Time           *metaV1.Time
		LoadTestStatus loadtestV1.LoadTestStatus
		Expected       bool
	}{
		{
			// Less than MaxWaitTimeForPods
			Time: &now,
			LoadTestStatus: loadtestV1.LoadTestStatus{
				Phase: loadtestV1.LoadTestCreating,
			},
			Expected: false,
		},
		{
			// pod was created MaxWaitTimeForPods and still in creation phase
			Time: &past,
			LoadTestStatus: loadtestV1.LoadTestStatus{
				Phase: loadtestV1.LoadTestCreating,
			},
			Expected: true,
		},
		{
			// Pod has been up for more than MaxWaitTimeForPods, but the test is running
			Time: &past,
			LoadTestStatus: loadtestV1.LoadTestStatus{
				Phase: loadtestV1.LoadTestRunning,
			},
			Expected: false,
		},
		{
			// Pod has been up for more than MaxWaitTimeForPods, but the test is running
			Time: nil,
			LoadTestStatus: loadtestV1.LoadTestStatus{
				Phase: loadtestV1.LoadTestCreating,
			},
			Expected: false,
		},
	}

	for _, test := range tests {
		r := workerPodHasTimeout(test.Time, test.LoadTestStatus)
		assert.Equal(t, test.Expected, r)
	}
}

func TestSetLoadTestDefaults(t *testing.T) {
	jm := &JMeter{
		loadTest: &loadtestV1.LoadTest{},
	}

	err := jm.SetDefaults()
	require.NoError(t, err)
	assert.Equal(t, loadtestV1.LoadTestCreating, jm.loadTest.Status.Phase)
}

func TestGetLoadTestPhaseFromJob(t *testing.T) {
	var testPhases = []struct {
		ExpectedPhase loadtestV1.LoadTestPhase
		JobStatus     batchV1.JobStatus
	}{
		{
			loadtestV1.LoadTestStarting,
			batchV1.JobStatus{
				Active: 0,
			},
		},
		{
			loadtestV1.LoadTestRunning,
			batchV1.JobStatus{
				Active: 1,
			},
		},
		{
			loadtestV1.LoadTestRunning,
			batchV1.JobStatus{
				Active: 1,
				Failed: 1,
			},
		},
		{
			loadtestV1.LoadTestFinished,
			batchV1.JobStatus{
				Active: 0,
				Failed: 2,
			},
		},
		{
			loadtestV1.LoadTestFinished,
			batchV1.JobStatus{
				Active:    0,
				Succeeded: 1,
				Failed:    0,
			},
		},
	}

	for _, test := range testPhases {
		phase := getLoadTestPhaseFromJob(test.JobStatus)
		assert.Equal(t, test.ExpectedPhase, phase)
	}
}
