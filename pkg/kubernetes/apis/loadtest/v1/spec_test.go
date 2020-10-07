package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildLoadTestSpec(t *testing.T) {
	var distributedPods int32 = 3

	type args struct {
		loadTestType    LoadTestType
		overwrite       bool
		distributedPods int32
		testFileStr     string
		testDataStr     string
		envVarsStr      string
		masterConfig    ImageDetails
		workerConfig    ImageDetails
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
				loadTestType:    "Fake",
				overwrite:       true,
				distributedPods: 3,
				testFileStr:     "something in the file",
				masterConfig: ImageDetails{
					Image: "image",
					Tag:   "tag",
				},
			},
			want: LoadTestSpec{
				Type:      "Fake",
				Overwrite: true,
				MasterConfig: ImageDetails{
					Image: "image",
					Tag:   "tag",
				},
				WorkerConfig:    ImageDetails{},
				DistributedPods: &distributedPods,
				TestFile:        "something in the file",
				TestData:        "",
				EnvVars:         "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSpec(tt.args.loadTestType, tt.args.overwrite, tt.args.distributedPods, tt.args.testFileStr, tt.args.testDataStr, tt.args.envVarsStr, tt.args.masterConfig, tt.args.workerConfig)
			assert.Equal(t, tt.want, got)
		})
	}
}
