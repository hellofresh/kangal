package jmeter

import (
	"testing"

	v1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"

	"github.com/stretchr/testify/assert"
)

func TestBuildJMeterLoadTestSpec(t *testing.T) {
	var distributedPods int32 = 3

	type args struct {
		overwrite       bool
		distributedPods int32
		tags            v1.LoadTestTags
		testFileStr     string
		testDataStr     string
		envVarsStr      string
	}
	tests := []struct {
		name    string
		args    args
		want    v1.LoadTestSpec
		wantErr bool
	}{
		{
			name: "Spec is valid",
			args: args{
				overwrite:       true,
				distributedPods: 3,
				tags:            v1.LoadTestTags{"team": "kangal"},
				testFileStr:     "something in the file",
				testDataStr:     "some test data",
			},
			want: v1.LoadTestSpec{
				Type:            "JMeter",
				Overwrite:       true,
				MasterConfig:    v1.ImageDetails{Image: masterImage, Tag: imageTag},
				WorkerConfig:    v1.ImageDetails{Image: workerImage, Tag: imageTag},
				DistributedPods: &distributedPods,
				Tags:            v1.LoadTestTags{"team": "kangal"},
				TestFile:        "something in the file",
				TestData:        "some test data",
				EnvVars:         "",
			},
			wantErr: false,
		},
		{
			name: "Spec invalid - invalid distributed pods",
			args: args{
				overwrite:       true,
				distributedPods: 0,
			},
			want:    v1.LoadTestSpec{},
			wantErr: true,
		},
		{
			name: "Spec invalid - require test file",
			args: args{
				overwrite:       true,
				distributedPods: 3,
			},
			want:    v1.LoadTestSpec{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildLoadTestSpec(tt.args.overwrite, tt.args.distributedPods, tt.args.tags, tt.args.testFileStr, tt.args.testDataStr, tt.args.envVarsStr)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want.Type, got.Type)
			assert.Equal(t, tt.want.Overwrite, got.Overwrite)
			assert.Equal(t, tt.want.MasterConfig, got.MasterConfig)
			assert.Equal(t, tt.want.WorkerConfig, got.WorkerConfig)
			assert.Equal(t, &tt.want.DistributedPods, &got.DistributedPods)
			assert.Equal(t, tt.want.Tags, got.Tags)
			assert.Equal(t, tt.want.TestFile, got.TestFile)
			assert.Equal(t, tt.want.TestData, got.TestData)
		})
	}
}
