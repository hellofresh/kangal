package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
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
			time.Hour * 2,
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
			time.Hour * 2,
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
			time.Hour * 2,
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
			time.Hour * 2,
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
			time.Hour * 2,
		},
	}

	for _, test := range testPhases {
		t.Run(test.Name, func(t *testing.T) {
			timedout := checkLoadTestLifeTimeExceeded(&test.LoadTest, test.Threshold)
			assert.Equal(t, test.ExpectedResponse, timedout)

		})
	}
}

func TestNewFileConfigMap(t *testing.T) {
	for _, ti := range []struct {
		tag         string
		cfgName     string
		filename    string
		content     []byte
		expected    *coreV1.ConfigMap
		expectError bool
	}{
		{
			tag:         "no configmap name",
			cfgName:     "",
			filename:    "file",
			content:     []byte("file content"),
			expected:    nil,
			expectError: true,
		},
		{
			tag:         "no filename",
			cfgName:     "test",
			filename:    "",
			content:     []byte("file content"),
			expected:    nil,
			expectError: true,
		},
		{
			tag:         "no content",
			cfgName:     "test",
			filename:    "file",
			content:     []byte{},
			expected:    nil,
			expectError: true,
		},
		{
			tag:      "valid args",
			cfgName:  "test",
			filename: "file",
			content:  []byte("file content"),
			expected: &coreV1.ConfigMap{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test",
				},
				BinaryData: map[string][]byte{
					"file": []byte("file content"),
				},
			},
			expectError: false,
		},
	} {
		t.Run(ti.tag, func(t *testing.T) {
			cfgmap, err := NewFileConfigMap(ti.cfgName, ti.filename, ti.content)

			assert.Equal(t, ti.expected, cfgmap)
			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
