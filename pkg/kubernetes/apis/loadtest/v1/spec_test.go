package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildLoadTestSpec(t *testing.T) {
	var distributedPods int32 = 3

	type args struct {
		loadTestType    LoadTestType
		overwrite       bool
		distributedPods int32
		tags            LoadTestTags
		testFileStr     string
		testDataStr     string
		envVars         map[string]string
		masterConfig    ImageDetails
		workerConfig    ImageDetails
		targetURL       string
		duration        time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    LoadTestSpec
		wantErr bool
	}{
		{
			name: "Spec is creating",
			args: args{
				loadTestType:    LoadTestTypeFake,
				overwrite:       true,
				distributedPods: 3,
				tags:            LoadTestTags{"team": "kangal"},
				testFileStr:     "something in the file",
				masterConfig: ImageDetails{
					Image: "image",
					Tag:   "tag",
				},
				envVars: map[string]string{"foo": "bar"},
			},
			want: LoadTestSpec{
				Type:      LoadTestTypeFake,
				Overwrite: true,
				MasterConfig: ImageDetails{
					Image: "image",
					Tag:   "tag",
				},
				WorkerConfig:    ImageDetails{},
				DistributedPods: &distributedPods,
				Tags:            LoadTestTags{"team": "kangal"},
				TestFile:        "something in the file",
				TestData:        "",
				EnvVars:         map[string]string{"foo": "bar"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSpec(tt.args.loadTestType, tt.args.overwrite, tt.args.distributedPods, tt.args.tags, tt.args.testFileStr, tt.args.testDataStr, tt.args.envVars, tt.args.masterConfig, tt.args.workerConfig, tt.args.targetURL, tt.args.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}
