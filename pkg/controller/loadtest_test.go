package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestShouldDeleteLoadtest(t *testing.T) {
	now := time.Now()
	metav1TimeTwoMonthsAgo := metaV1.NewTime(now.AddDate(0, -2, 0))
	metav1TimeNow := metaV1.NewTime(now)
	var testPhases = []struct {
		Name             string
		ExpectedResponse bool
		LoadTest         loadTestV1.LoadTest
		Threshold        time.Duration
	}{
		{
			"test finished long ago",
			true,
			loadTestV1.LoadTest{
				Status: loadTestV1.LoadTestStatus{
					Phase: loadTestV1.LoadTestFinished,
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
			loadTestV1.LoadTest{
				Status: loadTestV1.LoadTestStatus{
					Phase: loadTestV1.LoadTestFinished,
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
			loadTestV1.LoadTest{
				Status: loadTestV1.LoadTestStatus{
					Phase: loadTestV1.LoadTestErrored,
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
			loadTestV1.LoadTest{
				Status: loadTestV1.LoadTestStatus{
					Phase: loadTestV1.LoadTestErrored,
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
			loadTestV1.LoadTest{
				Status: loadTestV1.LoadTestStatus{
					Phase:     loadTestV1.LoadTestErrored,
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
