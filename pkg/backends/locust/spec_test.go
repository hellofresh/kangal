package locust

import (
	"testing"
	"time"

	"github.com/docker/distribution/reference"

	v1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"

	"github.com/stretchr/testify/assert"
)

func TestBuildLoadTestSpec(t *testing.T) {
	var distributedPods int32 = 3
	masterImageRef, _ := reference.ParseNormalizedNamed("alpine:3.2.1")

	type args struct {
		config          Config
		overwrite       bool
		distributedPods int32
		tags            v1.LoadTestTags
		testFileStr     string
		envVarsStr      string
		targetURL       string
		duration        time.Duration
		masterImageRef  reference.NamedTagged
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
				config:          Config{},
				overwrite:       true,
				distributedPods: 3,
				tags:            v1.LoadTestTags{"team": "kangal"},
				testFileStr:     "something in the file",
				envVarsStr:      "my-key,my-value",
				targetURL:       "http://my-app.my-domain.com",
				masterImageRef:  masterImageRef.(reference.NamedTagged),
			},
			want: v1.LoadTestSpec{
				Type:      "Locust",
				Overwrite: true,
				MasterConfig: v1.ImageDetails{
					Image: "docker.io/library/alpine",
					Tag:   "3.2.1",
				},
				DistributedPods: &distributedPods,
				Tags:            v1.LoadTestTags{"team": "kangal"},
				TestFile:        "something in the file",
				EnvVars:         "my-key,my-value",
				TargetURL:       "http://my-app.my-domain.com",
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
			got, err := BuildLoadTestSpec(tt.args.config, tt.args.overwrite, tt.args.distributedPods, tt.args.tags, tt.args.testFileStr, tt.args.envVarsStr, tt.args.targetURL, tt.args.duration, tt.args.masterImageRef)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want.Type, got.Type)
			assert.Equal(t, tt.want.Overwrite, got.Overwrite)
			assert.Equal(t, tt.want.MasterConfig, got.MasterConfig)
			assert.Equal(t, tt.want.MasterConfig, got.WorkerConfig)
			assert.Equal(t, &tt.want.DistributedPods, &got.DistributedPods)
			assert.Equal(t, tt.want.Tags, got.Tags)
			assert.Equal(t, tt.want.TestFile, got.TestFile)
			assert.Equal(t, tt.want.EnvVars, got.EnvVars)
			assert.Equal(t, tt.want.TargetURL, got.TargetURL)
		})
	}
}
