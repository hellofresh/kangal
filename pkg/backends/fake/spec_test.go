package fake

import (
	"testing"

	loadtestv1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"

	"github.com/stretchr/testify/assert"
)

func TestBuildFakeLoadTestSpec(t *testing.T) {
	var distributedPods int32 = 1

	type args struct {
		overwrite bool
	}
	tests := []struct {
		name    string
		args    args
		want    loadtestv1.LoadTestSpec
		wantErr bool
	}{
		{
			name: "Spec is valid",
			args: args{
				overwrite: true,
			},
			want: loadtestv1.LoadTestSpec{
				Type:      "Fake",
				Overwrite: true,
				MasterConfig: loadtestv1.ImageDetails{
					Image: sleepImage,
					Tag:   imageTag,
				},
				WorkerConfig:    loadtestv1.ImageDetails{},
				DistributedPods: &distributedPods,
				TestFile:        "",
				TestData:        "",
				EnvVars:         "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildLoadTestSpec(tt.args.overwrite)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
