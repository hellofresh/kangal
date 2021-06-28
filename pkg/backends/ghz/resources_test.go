package ghz

import (
	"testing"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestNewFileConfigMap(t *testing.T) {
	for _, ti := range []struct {
		tag         string
		cfgName     string
		filename    string
		content     string
		expected    *coreV1.ConfigMap
		expectError bool
	}{
		{
			tag:         "no configmap name",
			cfgName:     "",
			filename:    "file",
			content:     "file content",
			expected:    nil,
			expectError: true,
		},
		{
			tag:         "no filename",
			cfgName:     "test",
			filename:    "",
			content:     "file content",
			expected:    nil,
			expectError: true,
		},
		{
			tag:         "no content",
			cfgName:     "test",
			filename:    "file",
			content:     "",
			expected:    nil,
			expectError: true,
		},
		{
			tag:      "valid args",
			cfgName:  "test",
			filename: "file",
			content:  "file content",
			expected: &coreV1.ConfigMap{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test",
				},
				Data: map[string]string{
					"file": "file content",
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

func TestNewFileVolumeAndMount(t *testing.T) {
	for _, tt := range []struct {
		tag           string
		name          string
		cfg           string
		filename      string
		expectedVol   coreV1.Volume
		expectedMount coreV1.VolumeMount
	}{
		{
			tag:      "volume and mount are created with specified name, file and /data mount path",
			name:     "load-test-volume",
			cfg:      "test-configmap",
			filename: "testfile.json",
			expectedVol: coreV1.Volume{
				Name: "load-test-volume",
				VolumeSource: coreV1.VolumeSource{
					ConfigMap: &coreV1.ConfigMapVolumeSource{
						LocalObjectReference: coreV1.LocalObjectReference{
							Name: "test-configmap",
						},
					},
				},
			},
			expectedMount: coreV1.VolumeMount{
				Name:      "load-test-volume",
				MountPath: "/data/testfile.json",
				SubPath:   "testfile.json",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v, m := NewFileVolumeAndMount(tt.name, tt.cfg, tt.filename)
			assert.Equal(t, tt.expectedVol, v)
			assert.Equal(t, tt.expectedMount, m)
		})
	}
}
