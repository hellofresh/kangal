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
		testFileStr     string
		testDataStr     string
		envVarsStr      string
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
			name: "Spec is valid",
			args: args{
				loadTestType:    "Fake",
				overwrite:       true,
				distributedPods: 3,
				testFileStr:     "something in the file",
			},
			want: LoadTestSpec{
				Type:            "Fake",
				Overwrite:       true,
				MasterConfig:    ImageDetails{},
				WorkerConfig:    ImageDetails{},
				DistributedPods: &distributedPods,
				TestFile:        "something in the file",
				TestData:        "",
				EnvVars:         "",
			},
			wantErr: false,
		},
		{
			name: "Spec invalid - invalid load test type",
			args: args{
				loadTestType:    "Invalid Type",
				overwrite:       true,
				distributedPods: 3,
				testFileStr:     "something in the file",
			},
			want:    LoadTestSpec{},
			wantErr: true,
		},
		{
			name: "Spec invalid - invalid distributed pods",
			args: args{
				loadTestType:    "Fake",
				overwrite:       true,
				distributedPods: 0,
				testFileStr:     "something in the file",
			},
			want:    LoadTestSpec{},
			wantErr: true,
		},
		{
			name: "Spec invalid - invalid test file string",
			args: args{
				loadTestType:    "Fake",
				overwrite:       true,
				distributedPods: 3,
			},
			want:    LoadTestSpec{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildLoadTestSpec(tt.args.loadTestType, tt.args.overwrite, tt.args.distributedPods, tt.args.testFileStr, tt.args.testDataStr, tt.args.envVarsStr, tt.args.targetURL, tt.args.duration)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
