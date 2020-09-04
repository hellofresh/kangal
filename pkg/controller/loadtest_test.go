package controller

import (
	"testing"
	"time"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldDeleteLoadtest(t *testing.T) {
	now := time.Now()
	metav1TimeTwoMonthsAgo := metaV1.NewTime(now.AddDate(0, -2, 0))
	metav1TimeNow := metaV1.NewTime(now)
	var testPhases = []struct {
		Name             string
		ExpectedResponse bool
		LoadTest         loadtestV1.LoadTest
		Threshold        time.Duration
	}{
		{
			"test finished long ago",
			true,
			loadtestV1.LoadTest{
				Status: loadtestV1.LoadTestStatus{
					Phase: loadtestV1.LoadTestFinished,
					JobStatus: batchV1.JobStatus{
						CompletionTime: &metav1TimeTwoMonthsAgo,
					},
				},
			},
			time.Duration(time.Hour * 2),
		},
		{
			"test finished now",
			false,
			loadtestV1.LoadTest{
				Status: loadtestV1.LoadTestStatus{
					Phase: loadtestV1.LoadTestFinished,
					JobStatus: batchV1.JobStatus{
						CompletionTime: &metav1TimeNow,
					},
				},
			},
			time.Duration(time.Hour * 2),
		},
		{
			"test errored long ago",
			true,
			loadtestV1.LoadTest{
				Status: loadtestV1.LoadTestStatus{
					Phase: loadtestV1.LoadTestErrored,
					JobStatus: batchV1.JobStatus{
						CompletionTime: &metav1TimeTwoMonthsAgo,
					},
				},
			},
			time.Duration(time.Hour * 2),
		},
		{
			"test errored long ago, no completion",
			true,
			loadtestV1.LoadTest{
				Status: loadtestV1.LoadTestStatus{
					Phase: loadtestV1.LoadTestErrored,
					JobStatus: batchV1.JobStatus{
						CompletionTime: nil,
					},
				},
				ObjectMeta: metaV1.ObjectMeta{
					CreationTimestamp: metav1TimeTwoMonthsAgo,
				},
			},
			time.Duration(time.Hour * 2),
		},
		{
			"test errored now, no jobstatus",
			false,
			loadtestV1.LoadTest{
				Status: loadtestV1.LoadTestStatus{
					Phase:     loadtestV1.LoadTestErrored,
					JobStatus: batchV1.JobStatus{},
				},
				ObjectMeta: metaV1.ObjectMeta{
					CreationTimestamp: metav1TimeNow,
				},
			},
			time.Duration(time.Hour * 2),
		},
	}

	for _, test := range testPhases {
		t.Run(test.Name, func(t *testing.T) {
			timedout := checkLoadTestLifeTimeExceeded(&test.LoadTest, test.Threshold)
			assert.Equal(t, test.ExpectedResponse, timedout)

		})
	}
}
